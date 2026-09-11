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

// suspendScaledResource provides all the functions needed to scale a resource which is scaled by setting a suspend field.
type suspendScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// getSuspend gets the value of the suspend field on the workload
	getSuspend() (values.Replicas, values.Replicas)
	// setSuspend sets the value of the suspend field on the workload
	setSuspend(suspend bool)
	// getSavedResourcesRequests returns the saved CPU and memory requests for the workload based on the downscale replicas.
	getSavedResourcesRequests() *metrics.SavedResources
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// suspendScaledWorkload is a wrapper for all resources which are scaled by setting a suspend field.
type suspendScaledWorkload struct {
	suspendScaledResource
}

// LogUpscaleSuccessful logs a successful upscale using the original workload message style.
func (r *suspendScaledWorkload) LogUpscaleSuccessful(summary ScalingSummary, dryRun bool) {
	logWorkloadScalingMessage("unsuspended", "suspend", r, summary, dryRun)
}

// LogDownscaleSuccessful logs a successful downscale using the original workload message style.
func (r *suspendScaledWorkload) LogDownscaleSuccessful(summary ScalingSummary, dryRun bool) {
	logWorkloadScalingMessage("suspended", "suspend", r, summary, dryRun)
}

// GetChildren delegates child discovery to the wrapped resource when it supports ParentWorkload.
func (r *suspendScaledWorkload) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	parent, ok := r.suspendScaledResource.(ParentWorkload)
	if !ok {
		return nil, nil
	}

	children, err := parent.GetChildren(ctx, clientsets)
	if err != nil {
		return nil, fmt.Errorf("get children from parent workload: %w", err)
	}

	return children, nil
}

// ScaleUp scales up the underlying suspendScaledResource.
func (r *suspendScaledWorkload) ScaleUp() (ScalingSummary, error) {
	var summary ScalingSummary
	currentState, _ := r.getSuspend()

	originalState, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			slog.Debug("original replicas is not set, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
			return summary, nil
		}

		return summary, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalStateBool, err := originalState.AsBool()
	if err != nil {
		return summary, fmt.Errorf("failed to convert original state to bool: %w", err)
	}

	r.setSuspend(originalStateBool)

	removeOriginalReplicas(r)

	return ScalingSummary{IsUpdateNeeded: true, From: currentState, To: originalState}, nil
}

// ScaleDown scales down the underlying suspendScaledResource.
//

func (r *suspendScaledWorkload) ScaleDown(_ values.Replicas) (ScalingSummary, error) {
	currentState, targetScaleDownState := r.getSuspend()
	summary := ScalingSummary{
		SavedResources: metrics.NewSavedResources(0, 0),
		From:           currentState,
		To:             targetScaleDownState,
	}

	if currentState == targetScaleDownState {
		_, err := getOriginalReplicas(r)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if ok := errors.As(err, &originalReplicasUnsetErr); !ok {
				return summary, err
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

			return summary, nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

		savedResources := r.getSavedResourcesRequests()

		summary.SavedResources = savedResources

		return summary, nil
	}

	r.setSuspend(true)

	savedResources := r.getSavedResourcesRequests()

	setOriginalReplicas(currentState, r)

	summary.SavedResources = savedResources
	summary.IsUpdateNeeded = true

	return summary, nil
}
