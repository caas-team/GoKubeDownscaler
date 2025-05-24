// nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

func (p *podDisruptionBudget) AllowPercentageReplicas() bool {
	return true
}

// getMinAvailableInt returns the spec.MinAvailable value or an undefined/empty value.
//
//nolint:gocritic // unnamedResult: function returns unnamed result values intentionally
func (p *podDisruptionBudget) getMinAvailableInt() (int32, string) {
	minAvailable := p.Spec.MinAvailable
	if minAvailable == nil {
		return util.Undefined, util.EmptyString
	}

	if minAvailable.Type == intstr.String {
		return util.Undefined, minAvailable.StrVal
	}

	return minAvailable.IntVal, util.EmptyString
}

// setMinAvailable applies a new value to spec.MinAvailable.
func (p *podDisruptionBudget) setMinAvailable(targetMinAvailableInt int32, targetMinAvailableStr string) {
	var minAvailable intstr.IntOrString
	if targetMinAvailableStr != util.EmptyString {
		minAvailable = intstr.FromString(targetMinAvailableStr)
	} else {
		minAvailable = intstr.FromInt32(targetMinAvailableInt)
	}

	p.Spec.MinAvailable = &minAvailable
}

// getMaxUnavailableInt returns the spec.MaxUnavailable value or an undefined/empty value.
//
//nolint:gocritic // unnamedResult: function returns unnamed result values intentionally
func (p *podDisruptionBudget) getMaxUnavailableInt() (int32, string) {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return util.Undefined, util.EmptyString
	}

	if maxUnavailable.Type == intstr.String {
		return util.Undefined, maxUnavailable.StrVal
	}

	return maxUnavailable.IntVal, util.EmptyString
}

// setMaxUnavailable applies a new value to spec.MaxUnavailable.
func (p *podDisruptionBudget) setMaxUnavailable(targetMaxUnavailableInt int32, targetMaxUnavailableStr string) {
	var maxUnavailable intstr.IntOrString
	if targetMaxUnavailableStr != util.EmptyString {
		maxUnavailable = intstr.FromString(targetMaxUnavailableStr)
	} else {
		maxUnavailable = intstr.FromInt32(targetMaxUnavailableInt)
	}

	p.Spec.MaxUnavailable = &maxUnavailable
}

// ScaleUp scales the resource up.
func (p *podDisruptionBudget) ScaleUp() error {
	originalReplicasInt, originalReplicasStr, err := getOriginalReplicas(p)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetErr); ok {
			slog.Debug("original replicas is not set, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	maxUnavailable, _ := p.getMaxUnavailableInt()
	if maxUnavailable != util.Undefined {
		p.setMaxUnavailable(*originalReplicasInt, *originalReplicasStr)
		removeOriginalReplicas(p)

		return nil
	}

	minAvailable, _ := p.getMinAvailableInt()
	if minAvailable != util.Undefined {
		p.setMinAvailable(*originalReplicasInt, *originalReplicasStr)
		removeOriginalReplicas(p)

		return nil
	}

	return nil
}

// ScaleDown scales the resource down.
// nolint:cyclop // this function is too complex, but it is necessary to handle workload types. We should refactor this in the future.
func (p *podDisruptionBudget) ScaleDown(downscaleReplicas int32) error {
	maxUnavailableInt, maxUnavailableStr := p.getMaxUnavailableInt()
	if maxUnavailableInt != util.Undefined || maxUnavailableStr != util.EmptyString {
		if maxUnavailableInt == util.Undefined {
			trimmedMaxUnavailableStr := strings.TrimSuffix(maxUnavailableStr, "%")

			parsedInt, err := strconv.ParseInt(trimmedMaxUnavailableStr, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse percentage value from MaxUnavailable: %w", err)
			}

			maxUnavailableInt = int32(parsedInt)
		}

		if maxUnavailableInt == downscaleReplicas {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		p.setMaxUnavailable(downscaleReplicas, "")
		setOriginalReplicas(maxUnavailableInt, maxUnavailableStr, p)

		return nil
	}

	minAvailableInt, minAvailableStr := p.getMinAvailableInt()
	if minAvailableInt != util.Undefined || minAvailableStr != util.EmptyString {
		if minAvailableInt == util.Undefined {
			trimmedMinAvailableStr := strings.TrimSuffix(minAvailableStr, "%")

			parsedInt, err := strconv.ParseInt(trimmedMinAvailableStr, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse percentage value from MaxUnavailable: %w", err)
			}

			minAvailableInt = int32(parsedInt)
		}

		if minAvailableInt == downscaleReplicas {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		p.setMinAvailable(downscaleReplicas, "")
		setOriginalReplicas(minAvailableInt, minAvailableStr, p)

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
