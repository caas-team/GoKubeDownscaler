package scalable

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getPodDisruptionBudgets is the getResourceFunc for podDisruptionBudget
func getPodDisruptionBudgets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	poddisruptionbudgets, err := clientsets.Kubernetes.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get poddisruptionbudgets: %w", err)
	}
	for _, item := range poddisruptionbudgets.Items {
		results = append(results, &podDisruptionBudget{&item})
	}
	return results, nil
}

// podDisruptionBudget is a wrapper for policy/v1.podDisruptionBudget to implement the Workload interface
type podDisruptionBudget struct {
	*policy.PodDisruptionBudget
}

// getMinAvailableInt returns the spec.MinAvailable value if it is not a percentage
func (p *podDisruptionBudget) getMinAvailableInt() int32 {
	minAvailable := p.Spec.MinAvailable
	if minAvailable == nil {
		return values.Undefined
	}
	if minAvailable.Type == intstr.String {
		return values.Undefined
	}
	return minAvailable.IntVal
}

// setMinAvailable applies a new value to spec.MinAvailable
func (p *podDisruptionBudget) setMinAvailable(targetMinAvailable int32) error {
	minAvailable := intstr.FromInt32(targetMinAvailable)
	p.Spec.MinAvailable = &minAvailable
	return nil
}

// getMaxUnavailableInt returns the spec.MaxUnavailable value if it is not a percentage
func (p *podDisruptionBudget) getMaxUnavailableInt() int32 {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return values.Undefined
	}
	if maxUnavailable.Type == intstr.String {
		return values.Undefined
	}
	return maxUnavailable.IntVal
}

// setMaxUnavailable applies a new value to spec.MaxUnavailable
func (p *podDisruptionBudget) setMaxUnavailable(targetMaxUnavailable int32) error {
	maxUnavailable := intstr.FromInt32(targetMaxUnavailable)
	p.Spec.MaxUnavailable = &maxUnavailable
	return nil
}

// ScaleUp scales the resource up
func (p *podDisruptionBudget) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(p)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
		return nil
	}
	maxUnavailable := p.getMaxUnavailableInt()
	minAvailable := p.getMinAvailableInt()
	if maxUnavailable != values.Undefined {
		err = p.setMaxUnavailable(*originalReplicas)
		if err != nil {
			return fmt.Errorf("failed to set maxUnavailable for workload: %w", err)
		}
		removeOriginalReplicas(p)
		return nil
	}
	if minAvailable != values.Undefined {
		err = p.setMinAvailable(*originalReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		removeOriginalReplicas(p)
		return nil
	}
	slog.Debug("can't scale PodDisruptionBudgets with percent availability", "workload", p.GetName(), "namespace", p.GetNamespace())
	return nil
}

// ScaleDown scales the resource down
func (p *podDisruptionBudget) ScaleDown(downscaleReplicas int32) error {
	maxUnavailable := p.getMaxUnavailableInt()
	minAvailable := p.getMinAvailableInt()
	if maxUnavailable != values.Undefined {
		if maxUnavailable == downscaleReplicas {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}
		err := p.setMaxUnavailable(downscaleReplicas)
		if err != nil {
			return fmt.Errorf("failed to set maxUnavailable for workload: %w", err)
		}
		setOriginalReplicas(maxUnavailable, p)
		return nil
	}
	if minAvailable != values.Undefined {
		if minAvailable == downscaleReplicas {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}
		err := p.setMinAvailable(downscaleReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		setOriginalReplicas(minAvailable, p)
		return nil
	}
	slog.Debug("can't scale PodDisruptionBudgets with percent availability", "workload", p.GetName(), "namespace", p.GetNamespace())
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (p *podDisruptionBudget) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.PolicyV1().PodDisruptionBudgets(p.Namespace).Update(ctx, p.PodDisruptionBudget, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update poddisruptionbudget: %w", err)
	}
	return nil
}
