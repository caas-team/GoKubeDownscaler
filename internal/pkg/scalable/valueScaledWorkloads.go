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
	logUpscaleSuccessful(summary ScalingSummary, dryRun bool)
	logDownscaleSuccessful(summary ScalingSummary, dryRun bool)
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
func (v *valueScaledWorkload) LogUpscaleSuccessful(summary ScalingSummary, dryRun bool) {
	v.logUpscaleSuccessful(summary, dryRun)
}

// LogDownscaleSuccessful delegates resource-specific downscale logging.
func (v *valueScaledWorkload) LogDownscaleSuccessful(summary ScalingSummary, dryRun bool) {
	v.logDownscaleSuccessful(summary, dryRun)
}

// ScaleUp scales up the underlying valueScaledResource.
func (v *valueScaledWorkload) ScaleUp() (ScalingSummary, error) {
	var summary ScalingSummary

	currentState, _, err := v.getValue()
	if err != nil {
		return summary, fmt.Errorf("failed to get current value for workload: %w", err)
	}

	originalState, err := getOriginalReplicas(v)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			slog.Debug("original replicas is not set, skipping", "workload", v.GetName(), "namespace", v.GetNamespace())
			return summary, nil
		}

		return summary, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	err = v.setValue(originalState)
	if err != nil {
		return summary, fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(v)

	return ScalingSummary{IsUpdateNeeded: true, FromReplicas: currentState, ToReplicas: originalState}, nil
}

// ScaleDown scales down the underlying valueScaledResource.
func (v *valueScaledWorkload) ScaleDown(_ values.Replicas) (ScalingSummary, error) {
	currentState, targetScaleDownState, err := v.getValue()

	summary := ScalingSummary{SavedResources: metrics.NewSavedResources(0, 0), FromReplicas: currentState, ToReplicas: targetScaleDownState}
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

			slog.Debug("workload is already at target scale down state, skipping", "workload", v.GetName(), "namespace", v.GetNamespace())

			return summary, nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", v.GetName(), "namespace", v.GetNamespace())

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
