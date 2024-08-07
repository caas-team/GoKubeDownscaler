package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	annotationOriginalReplicas = "downscaler/original-replicas"
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
	originalReplicas, err := workload.GetCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == replicas {
		slog.Debug("workload is already at downscale replicas, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	workload.SetReplicas(replicas)
	err = c.setOriginalReplicas(originalReplicas, workload)
	if err != nil {
		return fmt.Errorf("failed to set original replicas annotation: %w", err)
	}
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled down workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// UpscaleWorkload upscales the workload to the original replicas
func (c client) UpscaleWorkload(workload scalable.Workload, ctx context.Context) error {
	currentReplicas, err := workload.GetCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
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
	err = c.removeOriginalReplicas(workload)
	if err != nil {
		return fmt.Errorf("failed to set original replicas annotation: %w", err)
	}
	err = workload.Update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	slog.Debug("successfully scaled up workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
	return nil
}

// setOriginalReplicas sets the original replicas annotation on the workload
func (c client) setOriginalReplicas(originalReplicas int, workload scalable.Workload) error {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[annotationOriginalReplicas] = fmt.Sprintf("%d", originalReplicas)
	workload.SetAnnotations(annotations)
	return nil
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
		return 0, fmt.Errorf("failed to get original replicas: %w", err)
	}
	return originalReplicas, nil
}

func (c client) removeOriginalReplicas(workload scalable.Workload) error {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
	return nil
}
