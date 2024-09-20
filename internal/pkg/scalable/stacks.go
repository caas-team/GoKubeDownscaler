package scalable

import (
	"context"
	"fmt"

	zalandov1 "github.com/zalando-incubator/stackset-controller/pkg/apis/zalando.org/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStacks is the getResourceFunc for Zalando Stacks
func getStacks(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	stacks, err := clientsets.Zalando.ZalandoV1().Stacks(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range stacks.Items {
		results = append(results, &replicaScaledWorkload{&stack{&item}})
	}
	return results, nil
}

// stack is a wrapper for zalando.org/v1.Stack to implement the replicaScaledResource interface
type stack struct {
	*zalandov1.Stack
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (s *stack) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource
func (s *stack) getReplicas() (int32, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return *s.Spec.Replicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *stack) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Zalando.ZalandoV1().Stacks(s.Namespace).Update(ctx, s.Stack, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}