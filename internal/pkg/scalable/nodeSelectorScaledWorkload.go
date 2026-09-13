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
func (r *nodeSelectorScaledWorkload) ScaleUp(logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}

	summary := scalingSummary{From: values.BooleanReplicas(true), To: values.BooleanReplicas(false)}

	updateNeeded, err := r.scaleUp(logger)
	if err != nil {
		return summary, err
	}

	summary.IsUpdateNeeded = updateNeeded

	return summary, nil
}

// ScaleDown scales down the underlying nodeSelectorScaledResource.
func (r *nodeSelectorScaledWorkload) ScaleDown(downscaleReplicas values.Replicas, logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}

	savedResources, updateNeeded, err := r.scaleDown(downscaleReplicas, logger)
	if err != nil {
		return scalingSummary{}, err
	}

	return scalingSummary{
		SavedResources: savedResources,
		IsUpdateNeeded: updateNeeded,
		From:           values.BooleanReplicas(false),
		To:             values.BooleanReplicas(true),
	}, nil
}

// LogUpscaleSuccessful logs a successful upscale using the node selector message style.
func (r *nodeSelectorScaledWorkload) LogUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("scaled up", "nodeSelector", r, summary, dryRun, logger)
}

// LogDownscaleSuccessful logs a successful downscale using the node selector message style.
func (r *nodeSelectorScaledWorkload) LogDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("scaled down", "nodeSelector", r, summary, dryRun, logger)
}

func (r *nodeSelectorScaledWorkload) scaleUp(logger *slog.Logger) (bool, error) {
	if logger == nil {
		logger = slog.Default()
	}

	_, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if errors.As(err, &originalReplicasUnsetErr) {
			logger.Debug("original replicas is not set, skipping")
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

func (r *nodeSelectorScaledWorkload) scaleDown(_ values.Replicas, logger *slog.Logger) (*metrics.SavedResources, bool, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if _, hasLabel := r.getNodeSelector()[labelMatchNone]; hasLabel {
		_, err := getOriginalReplicas(r)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if !errors.As(err, &originalReplicasUnsetErr) {
				return metrics.NewSavedResources(0, 0), false, fmt.Errorf("failed to get original replicas for workload: %w", err)
			}

			logger.Debug("workload is already at target scale down state, skipping")

			return metrics.NewSavedResources(0, 0), false, nil
		}

		logger.Debug("workload is already scaled down, skipping")

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
