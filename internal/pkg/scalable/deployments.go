package scalable

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Define the GVK for deployments
var deploymentGVK = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "Deployment",
}

// getDeployments is the getResourceFunc for Deployments
func getDeployments(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	deployments, err := clientsets.Kubernetes.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range deployments.Items {
		results = append(results, &replicaScaledWorkload{&deployment{&item}})
	}
	return results, nil
}

// deployment is a wrapper for apps/v1.Deployment to implement the replicaScaledResource interface
type deployment struct {
	*appsv1.Deployment
}

// GetObjectKind implements the scalableResource interface and sets the GVK
func (d *deployment) GetObjectKind() schema.ObjectKind {
	return d
}

// GroupVersionKind returns the GVK for deployments
func (d *deployment) GroupVersionKind() schema.GroupVersionKind {
	return deploymentGVK
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (d *deployment) setReplicas(replicas int32) error {
	d.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource
func (d *deployment) getReplicas() (int32, error) {
	replicas := d.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return *d.Spec.Replicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (d *deployment) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().Deployments(d.Namespace).Update(ctx, d.Deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
