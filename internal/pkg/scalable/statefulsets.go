//nolint:dupl // this code is very similar for every resource, but its not really abstractable to avoid more duplication
package scalable

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// regetStatefulSet is the regetResourceFunc for StatefulSets.
func regetStatefulSet(name, namespace string, clientsets *Clientsets, ctx context.Context) (Workload, error) {
	statefulset, err := clientsets.Kubernetes.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulset: %w", err)
	}

	return &replicaScaledWorkload{&statefulSet{statefulset}}, nil
}

// getStatefulSets is the getResourceFunc for StatefulSets.
func getStatefulSets(name, namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	if name != "" {
		results := make([]Workload, 0, 1)

		statefulset, err := clientsets.Kubernetes.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get statefulset: %w", err)
		}

		results = append(results, &replicaScaledWorkload{&statefulSet{statefulset}})

		return results, nil
	}

	statefulsets, err := clientsets.Kubernetes.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}

	results := make([]Workload, 0, len(statefulsets.Items))
	for i := range statefulsets.Items {
		results = append(results, &replicaScaledWorkload{&statefulSet{&statefulsets.Items[i]}})
	}

	return results, nil
}

// statefulset is a wrapper for apps/v1.StatefulSet to implement the replicaScaledResource interface.
type statefulSet struct {
	*appsv1.StatefulSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *statefulSet) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (s *statefulSet) getReplicas() (int32, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}

	return *s.Spec.Replicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *statefulSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}

	return nil
}
