package scalable

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getDeployments is the getResourceFunc for Deployments
func getDeployments(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]Workload, error) {
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

// SetReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
func (d deployment) SetReplicas(replicas int) {
	// #nosec G115
	newReplicas := int32(replicas)
	d.Spec.Replicas = &newReplicas
}

// GetCurrentReplicas gets the current amount of replicas of the resource
func (d deployment) GetCurrentReplicas() (int, error) {
	replicas := d.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*d.Spec.Replicas), nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (d deployment) Update(clientset *kubernetes.Clientset, ctx context.Context) error {
	_, err := clientset.AppsV1().Deployments(d.Namespace).Update(ctx, d.Deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
