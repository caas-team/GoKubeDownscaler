package scalable

import (
	"context"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStatefulSets is the getResourceFunc for StatefulSets.
func getStatefulSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
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

// statefulset is a wrapper for statefulset.v1.apps to implement the replicaScaledResource interface.
type statefulSet struct {
	*appsv1.StatefulSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *statefulSet) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (s *statefulSet) getReplicas() (values.Replicas, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(s.Kind, s.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (s *statefulSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	s.StatefulSet, err = clientsets.Kubernetes.AppsV1().StatefulSets(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the StatefulSet.
//
//nolint:nonamedreturns // using named return values for clarity and to simplify return statements
func (s *statefulSet) getSavedResourcesRequests(downscaleReplicas int32) (totalSavedCPU, totalSavedMemory float64) {
	for i := range s.Spec.Template.Spec.Containers {
		container := &s.Spec.Template.Spec.Containers[i]
		if container.Resources.Requests != nil {
			cpu := container.Resources.Requests.Cpu().AsApproximateFloat64()
			memory := container.Resources.Requests.Memory().AsApproximateFloat64()
			totalSavedCPU += cpu
			totalSavedMemory += memory
		}
	}

	totalSavedCPU *= totalSavedCPU * float64(*s.Spec.Replicas-downscaleReplicas)
	totalSavedMemory *= float64(*s.Spec.Replicas - downscaleReplicas)

	return totalSavedCPU, totalSavedMemory
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *statefulSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}

	return nil
}
