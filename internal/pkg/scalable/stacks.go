package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	zalandov1 "github.com/zalando-incubator/stackset-controller/pkg/apis/zalando.org/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStacks is the getResourceFunc for Zalando Stacks
func getStacks(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	deployments, err := clientsets.Zalando.ZalandoV1().Stacks(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range deployments.Items {
		results = append(results, &stack{&item})
	}
	return results, nil
}

// stack is a wrapper for zalando.org/v1.Stack to implement the Workload interface
type stack struct {
	*zalandov1.Stack
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (s *stack) setReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < 0 {
		return errBoundOnScalingTargetValue
	}

	// #nosec G115
	newReplicas := int32(replicas)
	s.Spec.Replicas = &newReplicas
	return nil
}

// getCurrentReplicas gets the current amount of replicas of the resource
func (s *stack) getCurrentReplicas() (int, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*s.Spec.Replicas), nil
}

// ScaleUp scales the resource up
func (s *stack) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(s)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}

	err = s.setReplicas(*originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}
	removeOriginalReplicas(s)
	return nil
}

// ScaleDown scales the resource down
func (s *stack) ScaleDown(downscaleReplicas int) error {
	originalReplicas, err := s.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already scaled down, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}

	err = s.setReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}
	setOriginalReplicas(originalReplicas, s)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *stack) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Zalando.ZalandoV1().Stacks(s.Namespace).Update(ctx, s.Stack, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
