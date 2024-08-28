package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	appsv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// getPodDisruptionBudgets is the getResourceFunc for podDisruptionBudget
func getPodDisruptionBudgets(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	poddisruptionbudgets, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
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
	*appsv1.PodDisruptionBudget
}

// getMinAvailableInt returns the spec.MinAvailable value if it is not a percentage
func (p *podDisruptionBudget) getMinAvailableInt() int {
	minAvailable := p.Spec.MinAvailable
	if minAvailable == nil {
		return values.Undefined
	}
	if minAvailable.Type == intstr.String {
		return values.Undefined
	}
	return int(minAvailable.IntVal)
}

// setMinAvailable applies a new value to spec.MinAvailable
func (p *podDisruptionBudget) setMinAvailable(targetMinAvailable int) error {
	if targetMinAvailable > math.MaxInt32 || targetMinAvailable < 0 {
		return errBoundOnScalingTargetValue
	}
	// #nosec G115
	p.Spec.MinAvailable = &intstr.IntOrString{IntVal: int32(targetMinAvailable), Type: intstr.Int}
	return nil
}

// getMaxUnavailableInt returns the spec.MaxUnavailable value if it is not a percentage
func (p *podDisruptionBudget) getMaxUnavailableInt() int {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return values.Undefined
	}
	if maxUnavailable.Type == intstr.String {
		return values.Undefined
	}
	return int(maxUnavailable.IntVal)
}

// setMaxUnavailable applies a new value to spec.MaxUnavailable
func (p *podDisruptionBudget) setMaxUnavailable(targetMaxUnavailable int) error {
	if targetMaxUnavailable > math.MaxInt32 || targetMaxUnavailable < 0 {
		return errBoundOnScalingTargetValue
	}
	// #nosec G115
	p.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: int32(targetMaxUnavailable), Type: intstr.Int}
	return nil
}

// ScaleUp upscale the resource when the downscale period ends
func (p *podDisruptionBudget) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(p)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == values.Undefined {
		slog.Debug("original replicas is not set, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
		return nil
	}
	maxUnavailable := p.getMaxUnavailableInt()
	minAvailable := p.getMinAvailableInt()
	if maxUnavailable != values.Undefined {
		err = p.setMaxUnavailable(originalReplicas)
		if err != nil {
			return fmt.Errorf("failed to set maxUnavailable for workload: %w", err)
		}
		removeOriginalReplicas(p)
		return nil
	}
	if minAvailable != values.Undefined {
		err = p.setMinAvailable(originalReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		removeOriginalReplicas(p)
		return nil
	}
	slog.Debug("can't scale PodDisruptionBudgets with percent availability", "workload", p.GetName(), "namespace", p.GetNamespace())
	return nil
}

// ScaleDown downscale the resource when the downscale period starts
func (p *podDisruptionBudget) ScaleDown(downscaleReplicas int) error {
	maxUnavailable := p.getMaxUnavailableInt()
	minAvailable := p.getMinAvailableInt()
	if maxUnavailable != values.Undefined {
		err := p.setMaxUnavailable(downscaleReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		setOriginalReplicas(maxUnavailable, p)
		return nil
	}
	if minAvailable != values.Undefined {
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
func (p *podDisruptionBudget) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.PolicyV1().PodDisruptionBudgets(p.Namespace).Update(ctx, p.PodDisruptionBudget, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update poddisruptionbudget: %w", err)
	}
	return nil
}
