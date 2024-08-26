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
		results = append(results, podDisruptionBudget{&item})
	}
	return results, nil
}

// podDisruptionBudget is a wrapper for policy/v1.podDisruptionBudget to implement the scalableResource interface
type podDisruptionBudget struct {
	*appsv1.PodDisruptionBudget
}

// getMinAvailableIfExistAndNotPercentageValue returns the spec.MinAvailable value if it is not a percentage
func (p podDisruptionBudget) getMinAvailableIfExistAndNotPercentageValue() (int32, bool, error) {
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

// setMinAvailable applies a new value to spec.MinAvailable
func (p podDisruptionBudget) setMinAvailable(targetMinAvailable int) error {
	if targetMinAvailable > math.MaxInt32 || targetMinAvailable < math.MinInt32 {
		return fmt.Errorf("targetMinAvailable exceeds int32 bounds")
	}

	// #nosec G115
	p.Spec.MinAvailable = &intstr.IntOrString{IntVal: int32(targetMinAvailable)}
	return nil
}

// getMaxUnavailableIfExistAndNotPercentageValue returns the spec.MaxUnavailable value if it is not a percentage
func (p podDisruptionBudget) getMaxUnavailableIfExistAndNotPercentageValue() (int32, bool, error) {
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

// setMaxUnavailable applies a new value to spec.MaxUnavailable
func (p podDisruptionBudget) setMaxUnavailable(targetMaxUnavailable int) error {
	if targetMaxUnavailable > math.MaxInt32 || targetMaxUnavailable < math.MinInt32 {
		return fmt.Errorf("targetMaxAvailable exceeds int32 bounds")
	}

	// #nosec G115
	p.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: int32(targetMaxUnavailable)}
	return nil
}

// ScaleUp upscale the resource when the downscale period ends
func (p podDisruptionBudget) ScaleUp() error {
	minAvailableValue, minAvailableExists, errMinAvailable := p.getMinAvailableIfExistAndNotPercentageValue()
	maxUnavailableValue, maxUnavailableExists, errMaxUnavailable := p.getMaxUnavailableIfExistAndNotPercentageValue()

	if errMinAvailable != nil {
		return fmt.Errorf("failed to get original minAvailable for workload: %w", errMinAvailable)
	}

	if errMaxUnavailable != nil {
		return fmt.Errorf("failed to get original maxUnavailable for workload: %w", errMaxUnavailable)
	}

	originalReplicas, err := getOriginalReplicas(p)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == values.Undefined {
		slog.Debug("original replicas is not set, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
		return nil
	}

	switch {
	case minAvailableExists:
		intMinAvailableValue := int(minAvailableValue)
		if originalReplicas == intMinAvailableValue {
			slog.Debug("workload is already at original values, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}
		err = p.setMinAvailable(originalReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		removeOriginalReplicas(p)
		return nil

	case maxUnavailableExists:
		intMaxUnavailableValue := int(maxUnavailableValue)
		if originalReplicas == intMaxUnavailableValue {
			slog.Debug("workload is already at original values, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}
		err = p.setMaxUnavailable(originalReplicas)
		if err != nil {
			return fmt.Errorf("failed to set maxUnavailable for workload: %w", err)
		}
		removeOriginalReplicas(p)
		return nil

	default:
		return fmt.Errorf("workload is already at max unavailable replicas")
	}
}

// ScaleDown downscale the resource when the downscale period starts
func (p podDisruptionBudget) ScaleDown(downscaleReplicas int) error {
	minAvailableValue, minAvailableExists, errMinAvailable := p.getMinAvailableIfExistAndNotPercentageValue()
	maxUnavailableValue, maxUnavailableExists, errMaxUnavailable := p.getMaxUnavailableIfExistAndNotPercentageValue()

	if errMinAvailable != nil {
		return fmt.Errorf("failed to get original minAvailable for workload: %w", errMinAvailable)
	}

	if errMaxUnavailable != nil {
		return fmt.Errorf("failed to get original maxUnavailable for workload: %w", errMaxUnavailable)
	}

	switch {
	case minAvailableExists:
		intMinAvailableValue := int(minAvailableValue)
		if intMinAvailableValue == downscaleReplicas {
			slog.Debug("workload is already at downscale values, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}
		err := p.setMinAvailable(downscaleReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		setOriginalReplicas(intMinAvailableValue, p)
		return nil

	case maxUnavailableExists:
		intMaxUnavailableValue := int(maxUnavailableValue)
		if intMaxUnavailableValue == downscaleReplicas {
			slog.Debug("workload is already at downscale values, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}
		err := p.setMaxUnavailable(downscaleReplicas)
		if err != nil {
			return fmt.Errorf("failed to set minAvailable for workload: %w", err)
		}
		setOriginalReplicas(intMaxUnavailableValue, p)
		return nil

	default:
		return fmt.Errorf("the workload does not have minimum available or max unavailable value for this policy workload")
	}
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (p podDisruptionBudget) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.PolicyV1().PodDisruptionBudgets(p.Namespace).Update(ctx, p.PodDisruptionBudget, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update poddisruptionbudget: %w", err)
	}
	return nil
}
