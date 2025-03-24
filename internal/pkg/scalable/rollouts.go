//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getRollouts is the getResourceFunc for Argo Rollouts.
func getRollouts(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	rollouts, err := clientsets.Argo.ArgoprojV1alpha1().Rollouts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get rollouts: %w", err)
	}

	results := make([]Workload, 0, len(rollouts.Items))
	for i := range rollouts.Items {
		results = append(results, &replicaScaledWorkload{&rollout{&rollouts.Items[i]}})
	}

	return results, nil
}

// parseRolloutFromAdmissionRequest parses the admission review and returns the rollout.
func parseRolloutFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var roll argov1alpha1.Rollout
	if err := json.Unmarshal(review.Request.Object.Raw, &roll); err != nil {
		return nil, fmt.Errorf("failed to decode Deployment: %w", err)
	}

	return &replicaScaledWorkload{&rollout{&roll}}, nil
}

// rollout is a wrapper for rollout.v1alpha1.argoproj.io to implement the replicaScaledResource interface.
type rollout struct {
	*argov1alpha1.Rollout
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (r *rollout) setReplicas(replicas int32) error {
	r.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (r *rollout) getReplicas() (values.Replicas, error) {
	replicas := r.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(r.Kind, r.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (r *rollout) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	r.Rollout, err = clientsets.Argo.ArgoprojV1alpha1().Rollouts(r.Namespace).Get(ctx, r.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get rollout: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (r *rollout) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Argo.ArgoprojV1alpha1().Rollouts(r.Namespace).Update(ctx, r.Rollout, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update rollout: %w", err)
	}

	return nil
}
