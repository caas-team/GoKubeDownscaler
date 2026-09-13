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

// valueScaledResource provides all the functions needed to scale a resource by setting a field to a particular value.
type valueScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// setValue sets the value of the key where downscaling is performed
	setValue(value values.Replicas) error
	// getValue gets the current value of the key where downscaling is performed and the value used for downscaling
	getValue() (values.Replicas, values.Replicas, error)
	// getSavedResourcesRequests returns the saved CPU and memory requests for the workload based on the downscale replicas.
	getSavedResourcesRequests() *metrics.SavedResources
	// logUpscaleSuccessful logs a successful upscale operations or dry-run upscale operations.
	logUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger)
	// logDownscaleSuccessful logs a successful downscale operations or dry-run downscale operations.
	logDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger)
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// valueScaledWorkload is a wrapper for all resources which are scaled by setting custom value field.
type valueScaledWorkload struct {
	valueScaledResource
}

// LogUpscaleSuccessful delegates resource-specific upscale logging.
func (v *valueScaledWorkload) LogUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	v.logUpscaleSuccessful(summary, dryRun, logger)
}

// LogDownscaleSuccessful delegates resource-specific downscale logging.
func (v *valueScaledWorkload) LogDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	v.logDownscaleSuccessful(summary, dryRun, logger)
}

// ScaleUp scales up the underlying valueScaledResource.
func (v *valueScaledWorkload) ScaleUp(logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var summary scalingSummary

	currentState, _, err := v.getValue()
	if err != nil {
		return summary, fmt.Errorf("failed to get current value for workload: %w", err)
	}

	originalState, err := getOriginalReplicas(v)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			logger.Debug("original replicas is not set, skipping")

			return summary, nil
		}

		return summary, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	err = v.setValue(originalState)
	if err != nil {
		return summary, fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(v)

	return scalingSummary{IsUpdateNeeded: true, From: currentState, To: originalState}, nil
}

// ScaleDown scales down the underlying valueScaledResource.
func (v *valueScaledWorkload) ScaleDown(_ values.Replicas, logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}

	currentState, targetScaleDownState, err := v.getValue()

	summary := scalingSummary{SavedResources: metrics.NewSavedResources(0, 0), From: currentState, To: targetScaleDownState}
	if err != nil {
		return summary, err
	}

	if currentState == targetScaleDownState {
		_, err = getOriginalReplicas(v)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if ok := errors.As(err, &originalReplicasUnsetErr); !ok {
				return summary, err
			}

			logger.Debug("workload is already at target scale down state, skipping")

			return summary, nil
		}

		logger.Debug("workload is already scaled down, skipping")

		return summary, nil
	}

	savedResources := v.getSavedResourcesRequests()

	err = v.setValue(targetScaleDownState)
	if err != nil {
		return summary, fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	setOriginalReplicas(currentState, v)

	summary.SavedResources = savedResources
	summary.IsUpdateNeeded = true

	return summary, nil
}
