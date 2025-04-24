// nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getPodDisruptionBudgets is the getResourceFunc for podDisruptionBudget.
func getPodDisruptionBudgets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	poddisruptionbudgets, err := clientsets.Kubernetes.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get poddisruptionbudgets: %w", err)
	}

	results := make([]Workload, 0, len(poddisruptionbudgets.Items))
	for i := range poddisruptionbudgets.Items {
		results = append(results, &podDisruptionBudget{&poddisruptionbudgets.Items[i]})
	}

	return results, nil
}

// podDisruptionBudget is a wrapper for poddisruptionbudget.v1.policy to implement the Workload interface.
type podDisruptionBudget struct {
	*policy.PodDisruptionBudget
}

// getMinAvailableInt returns the spec.MinAvailable value if it is not a percentage.
func (p *podDisruptionBudget) getMinAvailableInt() int32 {
	minAvailable := p.Spec.MinAvailable
	if minAvailable == nil {
		return util.Undefined
	}

	if minAvailable.Type == intstr.String {
		return util.Undefined
	}

	return minAvailable.IntVal
}

// setMinAvailable applies a new value to spec.MinAvailable.
func (p *podDisruptionBudget) setMinAvailable(targetMinAvailable int32) {
	minAvailable := intstr.FromInt32(targetMinAvailable)
	p.Spec.MinAvailable = &minAvailable
}

// getMaxUnavailableInt returns the spec.MaxUnavailable value if it is not a percentage.
func (p *podDisruptionBudget) getMaxUnavailableInt() int32 {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return util.Undefined
	}

	if maxUnavailable.Type == intstr.String {
		return util.Undefined
	}

	return maxUnavailable.IntVal
}

// setMaxUnavailable applies a new value to spec.MaxUnavailable.
func (p *podDisruptionBudget) setMaxUnavailable(targetMaxUnavailable int32) {
	maxUnavailable := intstr.FromInt32(targetMaxUnavailable)
	p.Spec.MaxUnavailable = &maxUnavailable
}

// ScaleUp scales the resource up.
func (p *podDisruptionBudget) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(p)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetErr); ok {
			slog.Debug("original replicas is not set, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	maxUnavailable := p.getMaxUnavailableInt()
	if maxUnavailable != util.Undefined {
		p.setMaxUnavailable(*originalReplicas)
		removeOriginalReplicas(p)

		return nil
	}

	minAvailable := p.getMinAvailableInt()
	if minAvailable != util.Undefined {
		p.setMinAvailable(*originalReplicas)
		removeOriginalReplicas(p)

		return nil
	}

	slog.Debug("can't scale PodDisruptionBudgets with percent availability", "workload", p.GetName(), "namespace", p.GetNamespace())

	return nil
}

// ScaleDown scales the resource down.
func (p *podDisruptionBudget) ScaleDown(downscaleReplicas int32) error {
	maxUnavailable := p.getMaxUnavailableInt()
	if maxUnavailable != util.Undefined {
		if maxUnavailable == downscaleReplicas {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		p.setMaxUnavailable(downscaleReplicas)
		setOriginalReplicas(maxUnavailable, p)

		return nil
	}

	minAvailable := p.getMinAvailableInt()
	if minAvailable != util.Undefined {
		if minAvailable == downscaleReplicas {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		p.setMinAvailable(downscaleReplicas)
		setOriginalReplicas(minAvailable, p)

		return nil
	}

	slog.Debug("can't scale PodDisruptionBudgets with percent availability", "workload", p.GetName(), "namespace", p.GetNamespace())

	return nil
}

// Reget regets the resource from the Kubernetes API.
func (p *podDisruptionBudget) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	p.PodDisruptionBudget, err = clientsets.Kubernetes.PolicyV1().PodDisruptionBudgets(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get poddisruptionbudget: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (p *podDisruptionBudget) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.PolicyV1().PodDisruptionBudgets(p.Namespace).Update(ctx, p.PodDisruptionBudget, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update poddisruptionbudget: %w", err)
	}

	return nil
}
