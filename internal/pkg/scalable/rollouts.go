package scalable

import (
	"context"
	"fmt"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getRollouts is the getResourceFunc for Argo Rollouts
func getRollouts(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	rollouts, err := clientsets.Argo.ArgoprojV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get rollouts: %w", err)
	}
	results := make([]Workload, 0, len(rollouts.Items))
	for _, item := range rollouts.Items {
		results = append(results, &replicaScaledWorkload{&rollout{&item}})
	}
	return results, nil
}

// rollout is a wrapper for argoproj.io/v1alpha1.Rollout to implement the replicaScaledResource interface
type rollout struct {
	*argov1alpha1.Rollout
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (r *rollout) setReplicas(replicas int32) error {
	r.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource
func (r *rollout) getReplicas() (int32, error) {
	replicas := r.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return *r.Spec.Replicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (r *rollout) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Argo.ArgoprojV1alpha1().Rollouts(r.Namespace).Update(ctx, r.Rollout, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update rollout: %w", err)
	}
	return nil
}
