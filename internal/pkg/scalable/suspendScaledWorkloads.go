package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	"k8s.io/apimachinery/pkg/types"
)

// suspendScaledResource provides all the functions needed to scale a resource which is scaled by setting a suspend field.
type suspendScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// Patch applies a patch to the resource.
	Patch(clientsets *Clientsets, patchType types.PatchType, patchData []byte, manageFields bool, ctx context.Context) error
	// getSuspend gets the value of the suspend field on the workload
	getSuspend() (values.Replicas, values.Replicas)
	// setSuspend sets the value of the suspend field on the workload
	setSuspend(suspend bool)
	// getSavedResourcesRequests returns the saved CPU and memory requests for the workload based on the downscale replicas.
	getSavedResourcesRequests() *metrics.SavedResources
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// GenerateJSONDiff compares the workload with another workload and returns the differences as a jsondiff.Patch
	GenerateJSONDiff(workloadCopy Workload) (jsondiff.Patch, error)
	// GeneratePatchDiff compares the workload with another workload and returns the JSON patch bytes.
	GeneratePatchDiff(workloadCopy Workload) ([]byte, error)
}

// suspendScaledWorkload is a wrapper for all resources which are scaled by setting a suspend field.
type suspendScaledWorkload struct {
	suspendScaledResource
}

// LogUpscaleSuccessful logs a successful upscale using the original workload message style.
func (r *suspendScaledWorkload) LogUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("unsuspended", "suspend", r, summary, dryRun, logger)
}

// LogDownscaleSuccessful logs a successful downscale using the original workload message style.
func (r *suspendScaledWorkload) LogDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("suspended", "suspend", r, summary, dryRun, logger)
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
func (r *suspendScaledWorkload) ScaleUp(logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var summary scalingSummary
	currentState, _ := r.getSuspend()

	originalState, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			logger.Debug("original replicas is not set, skipping")

			return summary, nil
		}

		return summary, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalStateBool, err := originalState.AsBool()
	if err != nil {
		return summary, fmt.Errorf("failed to convert original state to bool: %w", err)
	}

	workloadCopy, err := r.Copy()
	if err != nil {
		return summary, fmt.Errorf("failed to copy workload before scaling up: %w", err)
	}

	r.setSuspend(originalStateBool)

	removeOriginalReplicas(r)

	patchData, err := createPatchData(workloadCopy, r)
	if err != nil {
		return summary, err
	}

	return scalingSummary{IsUpdateNeeded: true, From: currentState, To: originalState, PatchData: patchData}, nil
}

// shouldSkipScaleDown checks if the workload is already at the target scale down state.
func (r *suspendScaledWorkload) shouldSkipScaleDown(
	logger *slog.Logger,
	currentState, targetScaleDownState values.Replicas,
) (bool, *metrics.SavedResources, error) {
	if currentState != targetScaleDownState {
		return false, nil, nil
	}

	_, err := getOriginalReplicas(r)
	if err == nil {
		logger.Debug("workload is already scaled down, skipping")
		return true, r.getSavedResourcesRequests(), nil
	}

	var originalReplicasUnsetErr *OriginalReplicasUnsetError
	if !errors.As(err, &originalReplicasUnsetErr) {
		return false, nil, err
	}

	logger.Debug("workload is already at target scale down state, skipping")

	return true, metrics.NewSavedResources(0, 0), nil
}

// ScaleDown scales down the underlying suspendScaledResource.
//

func (r *suspendScaledWorkload) ScaleDown(_ values.Replicas, logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if pending, ok := pendingServerSidePatchSummary(r); ok {
		return pending, nil
	}

	currentState, targetScaleDownState := r.getSuspend()
	summary := scalingSummary{
		SavedResources: metrics.NewSavedResources(0, 0),
		From:           currentState,
		To:             targetScaleDownState,
	}

	skip, savedResources, err := r.shouldSkipScaleDown(logger, currentState, targetScaleDownState)
	if err != nil {
		return summary, err
	}

	if skip {
		summary.SavedResources = savedResources
		return summary, nil
	}

	workloadCopy, err := r.Copy()
	if err != nil {
		return summary, fmt.Errorf("failed to copy workload before scaling down: %w", err)
	}

	r.setSuspend(true)

	savedResources = r.getSavedResourcesRequests()

	setOriginalReplicas(currentState, r)

	patchData, err := createPatchData(workloadCopy, r)
	if err != nil {
		return summary, err
	}

	summary.SavedResources = savedResources
	summary.IsUpdateNeeded = true
	summary.PatchData = patchData

	return summary, nil
}
