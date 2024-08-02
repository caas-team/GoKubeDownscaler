package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	annotationOriginalReplicas = "downscaler/original-replicas"
)

var (
	timeout                 int64 = 30
	errResourceNotSupported       = errors.New("error: specified rescource type is not supported")
)

type Client interface {
	GetWorkloads(namespaces []string, ctx context.Context) ([]ScalableResource, error)
	GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error)
}

// NewClient makes a new Client for the specified
func NewClient(kubeconfig string) (client, error) {
	var kubeclient client

	config, err := getConfig(kubeconfig)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get config for kubernetes: %w", err)
	}
	kubeclient.dynClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get dynamic client for kubernetes: %w", err)
	}
	kubeclient.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return kubeclient, fmt.Errorf("failed to get clientset for kubernetes: %w", err)
	}
	return kubeclient, nil
}

type client struct {
	dynClient *dynamic.DynamicClient
	clientset *kubernetes.Clientset
}

func (c client) GetNamespaceAnnotations(namespace string, ctx context.Context) (map[string]string, error) {
	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return ns.Annotations, nil
}

func (c client) GetWorkloads(namespaces []string, resourceTypes []string, ctx context.Context) ([]ScalableResource, error) {
	var results []ScalableResource
	for _, namespace := range namespaces {
		for _, resourceType := range resourceTypes {
			getWorkload, ok := scalableResources[resourceType]
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

func (c client) DownscaleWorkload(replicas int, workload ScalableResource, ctx context.Context) error {
	originalReplicas, err := workload.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == replicas {
		slog.Debug("workload is already at downscale replicas, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	err = workload.setReplicas(replicas)
	if err != nil {
		return fmt.Errorf("failed to scale workload: %w", err)
	}
	err = c.setOriginalReplicas(originalReplicas, workload)
	if err != nil {
		return fmt.Errorf("failed to set original replicas annotation: %w", err)
	}
	err = workload.update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	return nil
}

func (c client) UpscaleWorkload(workload ScalableResource, ctx context.Context) error {
	currentReplicas, err := workload.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	originalReplicas, err := c.getOriginalReplicas(workload)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}
	if *originalReplicas == currentReplicas {
		slog.Debug("workload is already at original replicas, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return nil
	}

	err = workload.setReplicas(*originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to scale workload: %w", err)
	}
	err = c.removeOriginalReplicas(workload)
	if err != nil {
		return fmt.Errorf("failed to set original replicas annotation: %w", err)
	}
	err = workload.update(c.clientset, ctx)
	if err != nil {
		return fmt.Errorf("failed to update workload: %w", err)
	}
	return nil
}

func (c client) setOriginalReplicas(originalReplicas int, workload ScalableResource) error {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[annotationOriginalReplicas] = fmt.Sprintf("%d", originalReplicas)
	workload.SetAnnotations(annotations)
	return nil
}

func (c client) getOriginalReplicas(workload ScalableResource) (*int, error) {
	annotations := workload.GetAnnotations()
	originalReplicasString, ok := annotations[annotationOriginalReplicas]
	if !ok {
		return nil, nil
	}
	originalReplicas, err := strconv.Atoi(originalReplicasString)
	if err != nil {
		return nil, fmt.Errorf("failed to get original replicas: %w", err)
	}
	return &originalReplicas, nil
}

func (c client) removeOriginalReplicas(workload ScalableResource) error {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
	return nil
}
