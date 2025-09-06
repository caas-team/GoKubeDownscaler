//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const PodDisruptionBudgetKind = "PodDisruptionBudget"

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

// parsePodDisruptionBudgetFromAdmissionRequest parses the admission review and returns the podDisruptionBudget wrapped in a Workload.
//
//nolint:ireturn //required for interface-based workflow
func parsePodDisruptionBudgetFromAdmissionRequest(rawObject []byte) (Workload, error) {
	var pdb policy.PodDisruptionBudget
	if err := json.Unmarshal(rawObject, &pdb); err != nil {
		return nil, fmt.Errorf("failed to decode Deployment: %w", err)
	}

	return &podDisruptionBudget{&pdb}, nil
}

// podDisruptionBudget is a wrapper for poddisruptionbudget.v1.policy to implement the Workload interface.
type podDisruptionBudget struct {
	*policy.PodDisruptionBudget
}

func (p *podDisruptionBudget) AllowPercentageReplicas() bool {
	return true
}

// getMinAvailable returns the spec.MinAvailable value or an undefined/empty value.
//
// nolint: gocritic // unnamedResult: function returns unnamed result values intentionally
func (p *podDisruptionBudget) getMinAvailable() values.Replicas {
	minAvailable := p.Spec.MinAvailable
	if minAvailable == nil {
		return nil
	}

	return values.NewReplicasFromIntOrStr(minAvailable)
}

// setMinAvailable applies a new value to spec.MinAvailable.
func (p *podDisruptionBudget) setMinAvailable(targetMinAvailable values.Replicas) {
	minAvailable := targetMinAvailable.AsIntStr()

	p.Spec.MinAvailable = &minAvailable
}

// getMaxUnavailable returns the spec.MaxUnavailable value or an undefined/empty value.
//
// nolint: gocritic // unnamedResult: function returns unnamed result values intentionally
func (p *podDisruptionBudget) getMaxUnavailable() values.Replicas {
	maxUnavailable := p.Spec.MaxUnavailable
	if maxUnavailable == nil {
		return nil
	}

	return values.NewReplicasFromIntOrStr(maxUnavailable)
}

// setMaxUnavailable applies a new value to spec.MaxUnavailable.
func (p *podDisruptionBudget) setMaxUnavailable(targetMaxUnavailable values.Replicas) {
	maxUnavailable := targetMaxUnavailable.AsIntStr()
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

	maxUnavailable := p.getMaxUnavailable()
	if maxUnavailable != nil {
		p.setMaxUnavailable(originalReplicas)
		removeOriginalReplicas(p)

		return nil
	}

	minAvailable := p.getMinAvailable()
	if minAvailable != nil {
		p.setMinAvailable(originalReplicas)
		removeOriginalReplicas(p)

		return nil
	}

	return nil
}

// ScaleDown scales the resource down.
// nolint:cyclop // this function is too complex, but it is necessary to handle workload types. We should refactor this in the future.
func (p *podDisruptionBudget) ScaleDown(downscaleReplicas values.Replicas) error {
	maxUnavailable := p.getMaxUnavailable()
	if maxUnavailable != nil {
		if maxUnavailable.String() == downscaleReplicas.String() {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		p.setMaxUnavailable(downscaleReplicas)
		setOriginalReplicas(maxUnavailable, p)

		return nil
	}

	minAvailable := p.getMinAvailable()
	if minAvailable != nil {
		if minAvailable.String() == downscaleReplicas.String() {
			slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
			return nil
		}

		p.setMinAvailable(downscaleReplicas)
		setOriginalReplicas(minAvailable, p)

		return nil
	}

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

// Copy creates a deep copy of the given Workload, which is expected to be a podDisruptionBudget.
//
//nolint:ireturn //required for interface-based workflow
func (p *podDisruptionBudget) Copy() (Workload, error) {
	if p.PodDisruptionBudget == nil {
		return nil, newNilUnderlyingObjectError(PodDisruptionBudgetKind)
	}

	copied := p.DeepCopy()

	return &podDisruptionBudget{PodDisruptionBudget: copied}, nil
}

// Compare compares two podDisruptionBudget resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (p *podDisruptionBudget) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	pdbCopy, ok := workloadCopy.(*podDisruptionBudget)
	if !ok {
		return nil, newExpectTypeGotTypeError((*podDisruptionBudget)(nil), workloadCopy)
	}

	if p.PodDisruptionBudget == nil || pdbCopy.PodDisruptionBudget == nil {
		return nil, newNilUnderlyingObjectError(PodDisruptionBudgetKind)
	}

	diff, err := jsondiff.Compare(p.PodDisruptionBudget, pdbCopy.PodDisruptionBudget)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError(PodDisruptionBudgetKind, err)
	}

	return diff, nil
}
