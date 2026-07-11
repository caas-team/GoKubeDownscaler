package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
)

const (
	labelMatchNone      = "downscaler/match-none"
	labelMatchNoneValue = "true"
)

// nodeSelectorScaledResource provides all the functions needed to scale a resource which is scaled by mutating its node selector.
type nodeSelectorScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource.
	Update(clientsets *Clientsets, ctx context.Context) error
	// getNodeSelector gets the node selector of the resource.
	getNodeSelector() map[string]string
	// setNodeSelector sets the node selector of the resource.
	setNodeSelector(nodeSelector map[string]string)
	// getResourcesRequests returns the saved CPU and memory requests for the resource based on the downscale replicas.
	getResourcesRequests(_ int32) *metrics.SavedResources
	// Copy creates a deep copy of the resource.
	Copy() (Workload, error)
	// Compare compares the resource with another resource and returns the differences as a jsondiff.Patch.
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// nodeSelectorScaledWorkload is a wrapper for all resources which are scaled by mutating their node selector.
type nodeSelectorScaledWorkload struct {
	nodeSelectorScaledResource
}

// ScaleUp scales up the underlying nodeSelectorScaledResource.
func (r *nodeSelectorScaledWorkload) ScaleUp() (bool, error) {
	_, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if errors.As(err, &originalReplicasUnsetErr) {
			slog.Debug("original replicas is not set, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
			return false, nil
		}

		return false, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	nodeSelector := r.getNodeSelector()
	delete(nodeSelector, labelMatchNone)
	r.setNodeSelector(nodeSelector)

	removeOriginalReplicas(r)

	return true, nil
}

// ScaleDown scales down the underlying nodeSelectorScaledResource.
func (r *nodeSelectorScaledWorkload) ScaleDown(_ values.Replicas) (*metrics.SavedResources, bool, error) {
	if _, hasLabel := r.getNodeSelector()[labelMatchNone]; hasLabel {
		_, err := getOriginalReplicas(r)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if !errors.As(err, &originalReplicasUnsetErr) {
				return metrics.NewSavedResources(0, 0), false, fmt.Errorf("failed to get original replicas for workload: %w", err)
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

			return metrics.NewSavedResources(0, 0), false, nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

		return r.getResourcesRequests(0), false, nil
	}

	nodeSelector := r.getNodeSelector()
	if nodeSelector == nil {
		nodeSelector = map[string]string{}
	}

	nodeSelector[labelMatchNone] = labelMatchNoneValue
	r.setNodeSelector(nodeSelector)

	savedResources := r.getResourcesRequests(0)

	setOriginalReplicas(values.BooleanReplicas(false), r)

	return savedResources, true, nil
}
