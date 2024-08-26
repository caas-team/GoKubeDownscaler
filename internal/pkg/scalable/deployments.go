package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	"k8s.io/client-go/dynamic"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getDeployments is the getResourceFunc for Deployments
func getDeployments(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range deployments.Items {
		results = append(results, deployment{&item})
	}
	return results, nil
}

// deployment is a wrapper for appsv1.Deployment to implement the scalableResource interface
type deployment struct {
	*appsv1.Deployment
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
func (d deployment) setReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < math.MinInt32 {
		return fmt.Errorf("replicas value exceeds int32 bounds")
	}

	// #nosec G115
	newReplicas := int32(replicas)
	d.Spec.Replicas = &newReplicas
	return nil
}

// getCurrentReplicas gets the current amount of replicas of the resource
func (d deployment) getCurrentReplicas() (int, error) {
	replicas := d.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*d.Spec.Replicas), nil
}

// ScaleUp upscale the resource when the downscale period ends
func (d deployment) ScaleUp() error {
	currentReplicas, err := d.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get current replicas for workload: %w", err)
	}
	originalReplicas, err := getOriginalReplicas(d)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == values.Undefined {
		slog.Debug("original replicas is not set, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())
		return nil
	}
	if originalReplicas == currentReplicas {
		slog.Debug("workload is already at original replicas, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())
		return nil
	}

	err = d.setReplicas(originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}
	removeOriginalReplicas(d)
	return nil
}

// ScaleDown downscale the resource when the downscale period starts
func (d deployment) ScaleDown(downscaleReplicas int) error {
	originalReplicas, err := d.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already at downscale replicas, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())
		return nil
	}

	err = d.setReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}
	setOriginalReplicas(originalReplicas, d)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (d deployment) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.AppsV1().Deployments(d.Namespace).Update(ctx, d.Deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
