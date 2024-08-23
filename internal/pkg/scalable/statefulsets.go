package scalable

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getStatefulSets is the getResourceFunc for StatefulSets
func getStatefulSets(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]Workload, error) {
	var results []Workload
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}
	for _, item := range statefulsets.Items {
		results = append(results, statefulset{&item})
	}
	return results, nil
}

// statefulset is a wrapper for appsv1.kubernetes to implement the scalableResource interface
type statefulset struct {
	*appsv1.StatefulSet
}

// SetReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
func (s statefulset) SetReplicas(replicas int) {
	newReplicas := int32(replicas)
	s.Spec.Replicas = &newReplicas
}

// GetCurrentReplicas gets the current amount of replicas of the resource
func (s statefulset) GetCurrentReplicas() (int, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*s.Spec.Replicas), nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s statefulset) Update(clientset *kubernetes.Clientset, ctx context.Context) error {
	_, err := clientset.AppsV1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}
	return nil
}
