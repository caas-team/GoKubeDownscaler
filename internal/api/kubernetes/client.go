package kubernetes

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	keda "github.com/kedacore/keda/v2/pkg/generated/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
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
	// addWorkloadEvent creates a new event on the workload
	addWorkloadEvent(eventType string, reason string, id string, message string, workload scalable.Workload, ctx context.Context) error
}

// NewClient makes a new Client
func NewClient(kubeconfig string) (client, error) {
	var kubeclient client
	var clientsets scalable.Clientsets

	config, err := getConfig(kubeconfig)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get config for kubernetes: %w", err)
	}
	// set qps and burst rate limiting options. See https://kubernetes.io/docs/reference/config-api/apiserver-eventratelimit.v1alpha1/
	config.QPS = 500    // available queries per second, when unused will fill the burst buffer
	config.Burst = 1000 // the max size of the buffer of queries
	clientsets.Kubernetes, err = kubernetes.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for kubernetes resources: %w", err)
	}
	clientsets.Keda, err = keda.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for keda resources: %w", err)
	}
	kubeclient.clientsets = &clientsets
	return kubeclient, nil
}

// client is a kubernetes client with downscaling specific functions
type client struct {
	clientsets *scalable.Clientsets
}

// GetNamespaceAnnotations gets the annotations of the workload's namespace
func (c client) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	ns, err := c.clientsets.Kubernetes.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return ns.Annotations, nil
}

// GetWorkloads gets all workloads of the specified resources for the specified namespaces
func (c client) GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]scalable.Workload, error) {
	var results []scalable.Workload
	if namespaces == nil {
		namespaces = append(namespaces, "")
	}
	for _, namespace := range namespaces {
		for _, resourceType := range resourceTypes {
			slog.Debug("getting workloads from resource type", "resourceType", resourceType)
			getWorkloads, ok := scalable.GetResource[strings.ToLower(resourceType)]
			if !ok {
				return nil, errResourceNotSupported
			}
			workloads, err := getWorkloads(namespace, c.clientsets, ctx)
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
	err := workload.ScaleDown(replicas)
	if err != nil {
		return fmt.Errorf("failed to set the workload into a scaled down state: %w", err)
	}
	err = workload.Update(c.clientsets, ctx)
	if err != nil {
		return fmt.Errorf("failed to update the workload: %w", err)
	}
	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// UpscaleWorkload upscales the workload to the original replicas
func (c client) UpscaleWorkload(workload scalable.Workload, ctx context.Context) error {
	err := workload.ScaleUp()
	if err != nil {
		return fmt.Errorf("failed to set the workload into a scaled up state: %w", err)
	}
	err = workload.Update(c.clientsets, ctx)
	if err != nil {
		return fmt.Errorf("failed to update the workload: %w", err)
	}
	slog.Debug("successfully scaled up workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// addWorkloadEvent creates or updates a new event on the workload
func (c client) addWorkloadEvent(eventType, reason, id, message string, workload scalable.Workload, ctx context.Context) error {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s.%s", id, message)))
	name := fmt.Sprintf("%s.%s.%x", workload.GetName(), reason, hash)
	eventsClient := c.clientsets.Kubernetes.CoreV1().Events(workload.GetNamespace())

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
	_, err := c.clientsets.Kubernetes.CoreV1().Events(workload.GetNamespace()).Create(ctx, &corev1.Event{
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
		Type:           eventType,
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
