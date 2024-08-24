package kubernetes

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	annotationOriginalReplicas = "downscaler/original-replicas"

	componentName = "kubedownscaler"
)

var errResourceNotSupported = errors.New("error: specified rescource type is not supported")

// Client is a interface representing a high-level client to get and modify kubernetes resources
type Client interface {
	// GetNamespaceAnnotations gets the annotations of the workload's namespace
	GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error)
	// GetWorkloads gets all workloads of the specified resources for the specified namespaces
	GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error)
	// DownscaleWorkload downscales the workload to the specified replicas
	DownscaleWorkload(replicas int, workload scalable.Workload, ctx context.Context) error
	// UpscaleWorkload upscales the workload to the original replicas
	UpscaleWorkload(workload scalable.Workload, ctx context.Context) error
	// AddErrorEvent creates a new event on the workload
	AddErrorEvent(reason string, id string, message string, workload scalable.Workload, ctx context.Context) error
}

// NewClient makes a new Client
func NewClient(kubeconfig string) (client, error) {
	var kubeclient client

	config, err := getConfig(kubeconfig)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get config for kubernetes: %w", err)
	}
	kubeclient.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for kubernetes: %w", err)
	}
	return kubeclient, nil
}

// client is a kubernetes client with downscaling specific functions
type client struct {
	clientset *kubernetes.Clientset
}

// GetNamespaceAnnotations gets the annotations of the workload's namespace
func (c client) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return ns.Annotations, nil
}

// GetWorkloads gets all workloads of the specified resources for the specified namespaces
func (c client) GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error) {
	var results []scalable.Workload
	for _, namespace := range namespaces {
		for _, resourceType := range resourceTypes {
			getWorkload, ok := scalable.GetResource[resourceType]
			if !ok {
				return nil, errResourceNotSupported
			}
			workloads, err := getWorkload(namespace, c.clientset, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get workloads: %w", err)
			}
			results = append(results, workloads...)
		}
	}

	return results, nil
}

// DownscaleWorkload downscales the workload to the specified replicas
func (c client) DownscaleWorkload(replicas int, workload scalable.Workload, ctx context.Context) error {
	switch w := workload.(type) {
	case scalable.BatchWorkload:
		return c.DownscaleBatchWorkload(w, ctx)

	case scalable.AppWorkload:
		return c.DownscaleAppWorkload(replicas, w, ctx)

	case scalable.DaemonWorkload:
		return c.DownscaleDaemonWorkload(w, ctx)

	default:
		return fmt.Errorf("failed to correctly identify the workload")
	}
}

// DownscaleAppWorkload downscales the batch workload to the original suspend state
func (c client) DownscaleAppWorkload(replicas int, workload scalable.AppWorkload, ctx context.Context) error {
	originalReplicas, err := workload.GetCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == replicas {
		slog.Debug("workload is already at downscale replicas, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	workload.SetReplicas(replicas)
	c.setOriginalReplicas(originalReplicas, workload)
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// DownscaleBatchWorkload downscales the app workload to the specified replicas
func (c client) DownscaleBatchWorkload(workload scalable.BatchWorkload, ctx context.Context) error {
	const suspend = true

	originalSuspend, err := workload.GetSuspend()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalSuspend == suspend {
		slog.Debug("workload is already downscaled, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	workload.SetSuspend(suspend)
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// DownscaleDaemonWorkload downscales the daemon workload
func (c client) DownscaleDaemonWorkload(workload scalable.DaemonWorkload, ctx context.Context) error {
	nodeSelectorExists, err := workload.NodeSelectorExists("kube-downscaler-non-existent", "true")
	if err != nil {
		return fmt.Errorf("failed to downscale the workload: %w. Make sure you didn't specify a kubedownscaler reserved node selector", err)
	}
	if nodeSelectorExists {
		slog.Debug("workload is already downscaled, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	workload.SetNodeSelector("kube-downscaler-non-existent", "true")
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// UpscaleWorkload upscales the workload to the original replicas
func (c client) UpscaleWorkload(workload scalable.Workload, ctx context.Context) error {
	switch w := workload.(type) {
	case scalable.BatchWorkload:
		return c.UpscaleBatchWorkload(w, ctx)

	case scalable.AppWorkload:
		return c.UpscaleAppWorkload(w, ctx)

	case scalable.DaemonWorkload:
		return c.UpscaleDaemonWorkload(w, ctx)

	default:
		return fmt.Errorf("failed to correctly identify the workload")
	}

}

// UpscaleBatchWorkload upscales the batch workload to the original suspend state
func (c client) UpscaleBatchWorkload(workload scalable.BatchWorkload, ctx context.Context) error {
	const suspend = false

	currentSuspend, err := workload.GetSuspend()
	if err != nil {
		return fmt.Errorf("failed to get current replicas for workload: %w", err)
	}
	if currentSuspend == suspend {
		slog.Debug("workload is already upscaled, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	workload.SetSuspend(suspend)
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled up workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// UpscaleAppWorkload upscales the app workload to the original replicas
func (c client) UpscaleAppWorkload(workload scalable.AppWorkload, ctx context.Context) error {
	currentReplicas, err := workload.GetCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get current replicas for workload: %w", err)
	}
	originalReplicas, err := c.getOriginalReplicas(workload)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == values.Undefined {
		slog.Debug("original replicas is not set, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}
	if originalReplicas == currentReplicas {
		slog.Debug("workload is already at original replicas, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	workload.SetReplicas(originalReplicas)
	c.removeOriginalReplicas(workload)
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled up workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// UpscaleDaemonWorkload upscales set daemon workload to the upscale state
func (c client) UpscaleDaemonWorkload(workload scalable.DaemonWorkload, ctx context.Context) error {
	nodeSelectorExists, err := workload.NodeSelectorExists("kube-downscaler-non-existent", "true")
	if err != nil {
		return fmt.Errorf("failed to upscale the workload: %w. Make sure you didn't specify a kubedownscaler reserved node selector", err)
	}
	if !nodeSelectorExists {
		slog.Debug("workload is already upscaled, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	err = workload.RemoveNodeSelector("kube-downscaler-non-existent")
	if err != nil {
		return fmt.Errorf("failed to remove node selector from the workload: %w", err)
	}
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled up workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// setOriginalReplicas sets the original replicas annotation on the workload
func (c client) setOriginalReplicas(originalReplicas int, workload scalable.Workload) {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[annotationOriginalReplicas] = fmt.Sprintf("%d", originalReplicas)
	workload.SetAnnotations(annotations)
}

// getOriginalReplicas gets the original replicas annotation on the workload. nil is undefined
func (c client) getOriginalReplicas(workload scalable.Workload) (int, error) {
	annotations := workload.GetAnnotations()
	originalReplicasString, ok := annotations[annotationOriginalReplicas]
	if !ok {
		return values.Undefined, nil
	}
	originalReplicas, err := strconv.Atoi(originalReplicasString)
	if err != nil {
		return 0, fmt.Errorf("failed to parse original replicas annotation on workload: %w", err)
	}
	return originalReplicas, nil
}

func (c client) removeOriginalReplicas(workload scalable.Workload) {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
}

// AddErrorEvent creates or updates a new event on the workload
func (c client) AddErrorEvent(reason, id, message string, workload scalable.Workload, ctx context.Context) error {
	hash := sha1.Sum([]byte(fmt.Sprintf("%s.%s", id, message)))
	name := fmt.Sprintf("%s.%s.%.3x", workload.GetName(), reason, hash)
	eventsClient := c.clientset.CoreV1().Events(workload.GetNamespace())

	// check if event already exists
	if event, err := eventsClient.Get(ctx, name, metav1.GetOptions{}); err == nil && event != nil {
		// update event
		event.Count += 1
		event.LastTimestamp = metav1.Now()
		_, err := eventsClient.Update(ctx, event, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update event: %w", err)
		}
		return nil
	}

	// create event
	_, err := c.clientset.CoreV1().Events(workload.GetNamespace()).Create(ctx, &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: workload.GetNamespace(),
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       workload.GetObjectKind().GroupVersionKind().Kind,
			Namespace:  workload.GetNamespace(),
			Name:       workload.GetName(),
			UID:        workload.GetUID(),
			APIVersion: workload.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		},
		Reason:         reason,
		Message:        message,
		Type:           corev1.EventTypeWarning,
		Source:         corev1.EventSource{Component: componentName},
		FirstTimestamp: metav1.Now(),
		LastTimestamp:  metav1.Now(),
		Count:          1,
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}
	return nil
}
