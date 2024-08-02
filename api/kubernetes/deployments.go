package kubernetes

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func getDeployments(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]ScalableResource, error) {
	var results []ScalableResource
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range deployments.Items {
		results = append(results, deployment{&item})
	}
	return results, nil
}

type deployment struct {
	*appsv1.Deployment
}

func (d deployment) setReplicas(replicas int) error {
	newReplicas := int32(replicas)
	d.Spec.Replicas = &newReplicas
	return nil
}

func (d deployment) getCurrentReplicas() (int, error) {
	replicas := d.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*d.Spec.Replicas), nil
}

func (d deployment) update(clientset *kubernetes.Clientset, ctx context.Context) error {
	_, err := clientset.AppsV1().Deployments(d.Namespace).Update(ctx, d.Deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
