package scalable

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Define the GVK for statefulsets
var statefulSetGVK = schema.GroupVersionKind{
	Group:   "apps",
	Version: "v1",
	Kind:    "StatefulSet",
}

// getStatefulSets is the getResourceFunc for StatefulSets
func getStatefulSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	statefulsets, err := clientsets.Kubernetes.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}
	for _, item := range statefulsets.Items {
		results = append(results, &replicaScaledWorkload{&statefulSet{&item}})
	}
	return results, nil
}

// statefulset is a wrapper for apps/v1.StatefulSet to implement the replicaScaledResource interface
type statefulSet struct {
	*appsv1.StatefulSet
}

// GetObjectKind sets the GVK for statefulSet
func (s *statefulSet) GetObjectKind() schema.ObjectKind {
	return s
}

// GroupVersionKind returns the GVK for statefulSet
func (s *statefulSet) GroupVersionKind() schema.GroupVersionKind {
	return statefulSetGVK
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (s *statefulSet) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource
func (s *statefulSet) getReplicas() (int32, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return *s.Spec.Replicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *statefulSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}
	return nil
}
