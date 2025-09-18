//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	zalandov1 "github.com/zalando-incubator/stackset-controller/pkg/apis/zalando.org/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStacks is the getResourceFunc for Zalando Stacks.
func getStacks(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	stacks, err := clientsets.Zalando.ZalandoV1().Stacks(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get stacks: %w", err)
	}

	results := make([]Workload, 0, len(stacks.Items))
	for i := range stacks.Items {
		results = append(results, &replicaScaledWorkload{&stack{&stacks.Items[i]}})
	}

	return results, nil
}

// stack is a wrapper for stack.v1.zalando.org to implement the replicaScaledResource interface.
type stack struct {
	*zalandov1.Stack
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *stack) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (s *stack) getReplicas() (values.Replicas, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(s.Kind, s.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (s *stack) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	s.Stack, err = clientsets.Zalando.ZalandoV1().Stacks(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get stack: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the Stack.
//

func (s *stack) getSavedResourcesRequests(diffReplicas int32) *SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range s.Spec.PodTemplate.Spec.Containers {
		container := &s.Spec.PodTemplate.Spec.Containers[i]
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	totalSavedCPU *= float64(diffReplicas)
	totalSavedMemory *= float64(diffReplicas)

	return NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *stack) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Zalando.ZalandoV1().Stacks(s.Namespace).Update(ctx, s.Stack, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update stack: %w", err)
	}

	return nil
}
