package scalable

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// getDaemonSets is the getResourceFunc for DaemonSets
func getPodDisruptionBudgets(namespace string, clientset *kubernetes.Clientset, ctx context.Context) ([]Workload, error) {
	var results []Workload
	poddisruptionbudgets, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get poddisruptionbudgets: %w", err)
	}
	for _, item := range poddisruptionbudgets.Items {
		results = append(results, PodDisruptionBudget{&item})
	}
	return results, nil
}

// DaemonSet is a wrapper for batch/v1.CronJob to implement the scalableResource interface
type PodDisruptionBudget struct {
	*appsv1.PodDisruptionBudget
}

func (p PodDisruptionBudget) GetMinAvailableIfExistAndNotPercentageValue() (int32, bool, error) {
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

func (p PodDisruptionBudget) SetMinAvailable(targetMinAvailable int) {
	p.Spec.MinAvailable = &intstr.IntOrString{IntVal: int32(targetMinAvailable)}
}

func (p PodDisruptionBudget) GetMaxUnavailableIfExistAndNotPercentageValue() (int32, bool, error) {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return 0, false, nil
	}

	switch maxUnavailable.Type {
	case intstr.Int:
		// Directly return the integer value
		return maxUnavailable.IntVal, true, nil

	case intstr.String:
		// Handle the case where the value is a string
		return 0, false, fmt.Errorf("minAvailable is a string value and cannot be converted to int directly")

	default:
		// Handle unexpected types
		return 0, false, fmt.Errorf("unknown type for minAvailable")
	}
}

func (p PodDisruptionBudget) SetMaxUnavailable(targetMaxUnavailable int) {
	p.Spec.MaxUnavailable = &intstr.IntOrString{IntVal: int32(targetMaxUnavailable)}
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (p PodDisruptionBudget) Update(clientset *kubernetes.Clientset, ctx context.Context) error {
	_, err := clientset.PolicyV1().PodDisruptionBudgets(p.Namespace).Update(ctx, p.PodDisruptionBudget, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update poddisruptionbudget: %w", err)
	}
	return nil
}
