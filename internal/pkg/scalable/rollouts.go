package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getRollouts is the getResourceFunc for Argo Rollouts
func getRollouts(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	deployments, err := clientsets.Argo.ArgoprojV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range deployments.Items {
		results = append(results, &rollout{&item})
	}
	return results, nil
}

// rollout is a wrapper for argoproj.io/v1alpha1.Rollout to implement the Workload interface
type rollout struct {
	*argov1alpha1.Rollout
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
func (r *rollout) setReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < 0 {
		return errBoundOnScalingTargetValue
	}

	// #nosec G115
	newReplicas := int32(replicas)
	r.Spec.Replicas = &newReplicas
	return nil
}

// getCurrentReplicas gets the current amount of replicas of the resource
func (r *rollout) getCurrentReplicas() (int, error) {
	replicas := r.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*r.Spec.Replicas), nil
}

// ScaleUp scales the resource up
func (r *rollout) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(r)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
		return nil
	}

	err = r.setReplicas(*originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}
	removeOriginalReplicas(r)
	return nil
}

// ScaleDown scales the resource down
func (r *rollout) ScaleDown(downscaleReplicas int) error {
	originalReplicas, err := r.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
		return nil
	}

	err = r.setReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}
	setOriginalReplicas(originalReplicas, r)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (r *rollout) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Argo.ArgoprojV1alpha1().Rollouts(r.Namespace).Update(ctx, r.Rollout, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
