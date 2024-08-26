package scalable

import (
	"context"
	"fmt"
	"math"

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
		results = append(results, podDisruptionBudget{&item})
	}
	return results, nil
}

// podDisruptionBudget is a wrapper for policy/v1.podDisruptionBudget to implement the scalableResource interface
type podDisruptionBudget struct {
	*appsv1.PodDisruptionBudget
}

// GetMinAvailableIfExistAndNotPercentageValue returns the spec.MinAvailable value if it is not a percentage
func (p podDisruptionBudget) GetMinAvailableIfExistAndNotPercentageValue() (int32, bool, error) {
	minAvailable := p.Spec.MinAvailable
	if minAvailable == nil {
		return 0, false, nil
	}

	switch minAvailable.Type {
	case intstr.Int:
		// Directly return the integer value
		return minAvailable.IntVal, true, nil

	case intstr.String:
		// Handle the case where the value is a string
		return 0, false, fmt.Errorf("minAvailable is a string value and cannot be converted to int directly")

	default:
		// Handle unexpected types
		return 0, false, fmt.Errorf("unknown type for minAvailable")
	}
}

// SetMinAvailable applies a new value to spec.MinAvailable
func (p *podDisruptionBudget) SetMinAvailable(targetMinAvailable int) error {
	if targetMinAvailable > math.MaxInt32 || targetMinAvailable < math.MinInt32 {
		return fmt.Errorf("targetMinAvailable exceeds int32 bounds")
	}

	// #nosec G115
	p.Spec.MinAvailable = &intstr.IntOrString{IntVal: int32(targetMinAvailable)}
	return nil
}

// GetMaxUnavailableIfExistAndNotPercentageValue returns the spec.MaxUnavailable value if it is not a percentage
func (p podDisruptionBudget) GetMaxUnavailableIfExistAndNotPercentageValue() (int32, bool, error) {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return 0, false, nil
	}

	switch maxUnavailable.Type {
	case intstr.Int:
		// Directly return the integer value
		return maxUnavailable.IntVal, true, nil

	case intstr.String:
		// Handle the case where the value is a string (percentage)
		return 0, false, fmt.Errorf("minAvailable is a string value and cannot be converted to int directly")

	default:
		// Handle unexpected types
		return 0, false, fmt.Errorf("unknown type for minAvailable")
	}
}

// SetMaxUnavailable applies a new value to spec.MaxUnavailable
func (p podDisruptionBudget) SetMaxUnavailable(targetMaxUnavailable int) error {
	if targetMaxUnavailable > math.MaxInt32 || targetMaxUnavailable < math.MinInt32 {
		return fmt.Errorf("targetMaxAvailable exceeds int32 bounds")
	}

	// #nosec G115
	p.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: int32(targetMaxUnavailable)}
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (p podDisruptionBudget) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.PolicyV1().PodDisruptionBudgets(p.Namespace).Update(ctx, p.PodDisruptionBudget, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update poddisruptionbudget: %w", err)
	}
	return nil
}
