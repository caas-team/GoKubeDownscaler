package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	"k8s.io/apimachinery/pkg/types"
)

// replicaScaledResource provides all the functions needed to scale a resource which is scaled by setting the replica count.
type replicaScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// Patch applies a patch to the resource.
	Patch(clientsets *Clientsets, patchType types.PatchType, patchData []byte, manageFields bool, ctx context.Context) error
	// setReplicas sets the replicas of the workload
	setReplicas(replicas int32) error
	// getReplicas gets the replicas of the workload
	getReplicas() (values.Replicas, error)
	// getSavedResourcesRequests returns the saved CPU and memory requests for the workload based on the downscale replicas.
	getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// replicaScaledWorkload is a wrapper for all resources which are scaled by setting the replica count.
type replicaScaledWorkload struct {
	replicaScaledResource
}

// LogUpscaleSuccessful logs a successful upscale using the original workload message style.
func (r *replicaScaledWorkload) LogUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("scaled up", "replicas", r, summary, dryRun, logger)
}

// LogDownscaleSuccessful logs a successful downscale using the original workload message style.
func (r *replicaScaledWorkload) LogDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("scaled down", "replicas", r, summary, dryRun, logger)
}

// ScaleUp scales up the underlying replicaScaledResource.
func (r *replicaScaledWorkload) ScaleUp(logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var summary scalingSummary

	currentReplicas, err := r.getReplicas()
	if err != nil {
		return summary, fmt.Errorf("failed to get current replicas for workload: %w", err)
	}

	originalReplicas, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetErr); ok {
			logger.Debug("original replicas is not set, skipping")

			return summary, nil
		}

		return summary, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalReplicasInt32, err := originalReplicas.AsInt32()
	if err != nil {
		return summary, fmt.Errorf("failed to convert original replicas to int32: %w", err)
	}

	workloadCopy, err := r.Copy()
	if err != nil {
		return summary, fmt.Errorf("failed to copy workload before scaling up: %w", err)
	}

	err = r.setReplicas(originalReplicasInt32)
	if err != nil {
		return summary, fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(r)

	patchData, err := createPatchData(workloadCopy, r)
	if err != nil {
		return summary, err
	}

	return scalingSummary{IsUpdateNeeded: true, From: currentReplicas, To: originalReplicas, PatchData: patchData}, nil
}

// ScaleDown scales down the underlying replicaScaledResource.
//
// nolint:cyclop // refactoring this function would make it less readable and more fragmented
func (r *replicaScaledWorkload) ScaleDown(downscaleReplicas values.Replicas, logger *slog.Logger) (scalingSummary, error) {
	if logger == nil {
		logger = slog.Default()
	}

	downscaleReplicasInt32, err := downscaleReplicas.AsInt32()

	summary := scalingSummary{SavedResources: metrics.NewSavedResources(0, 0)}
	if err != nil {
		return summary, fmt.Errorf("failed to convert replicas to int32: %w", err)
	}

	currentReplicas, err := r.getReplicas()
	if err != nil {
		return summary, fmt.Errorf("failed to get current replicas for workload: %w", err)
	}

	currentReplicasInt32, err := currentReplicas.AsInt32()
	if err != nil {
		return summary, fmt.Errorf("failed to convert current replicas to int32: %w", err)
	}

	skip, summary, err := r.shouldSkipScaleDown(currentReplicas, currentReplicasInt32, downscaleReplicas, downscaleReplicasInt32, logger)
	if err != nil {
		return summary, err
	}

	if skip {
		return summary, nil
	}

	workloadCopy, err := r.Copy()
	if err != nil {
		return summary, fmt.Errorf("failed to copy workload before scaling down: %w", err)
	}

	err = r.setReplicas(downscaleReplicasInt32)
	if err != nil {
		return summary, fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	summary.SavedResources = r.getSavedResourcesRequests(currentReplicasInt32 - downscaleReplicasInt32)

	setOriginalReplicas(currentReplicas, r)

	patchData, err := createPatchData(workloadCopy, r)
	if err != nil {
		return summary, err
	}

	summary.IsUpdateNeeded = true
	summary.PatchData = patchData
	summary.From = currentReplicas
	summary.To = downscaleReplicas

	return summary, nil
}

// shouldSkipScaleDown checks if the workload is already at or below the target scale down replicas and returns a boolean
// indicating whether to skip the downscale, along with a scalingSummary and an error if any.
func (r *replicaScaledWorkload) shouldSkipScaleDown(
	currentReplicas values.Replicas,
	currentReplicasInt32 int32,
	downscaleReplicas values.Replicas,
	downscaleReplicasInt32 int32,
	logger *slog.Logger,
) (bool, scalingSummary, error) {
	summary := scalingSummary{SavedResources: metrics.NewSavedResources(0, 0)}

	// util.Undefined (-1) is a sentinel for "no current replicas set" (e.g. a ScaledObject without a
	// paused-replicas annotation). It must not be treated as already being at or below the downtime target,
	// otherwise such workloads would never be scaled down.
	if currentReplicasInt32 == util.Undefined || currentReplicasInt32 > downscaleReplicasInt32 {
		return false, summary, nil
	}

	originalReplicasInt32, isOriginalReplicasSet, err := getOriginalReplicasInt32(r)
	if err != nil {
		return false, summary, err
	}

	if !isOriginalReplicasSet {
		logger.Debug("workload is at or below target scale down replicas, skipping")

		summary.From = currentReplicas
		summary.To = downscaleReplicas

		return true, summary, nil
	}

	summary.SavedResources = r.getSavedResourcesRequests(originalReplicasInt32 - downscaleReplicasInt32)

	logger.Debug("workload is already scaled down, skipping")

	summary.From = currentReplicas
	summary.To = downscaleReplicas

	return true, summary, nil
}

// getOriginalReplicas retrieves the original replicas from the workload.
//
//nolint:nonamedreturns // using named return values for clarity and to simplify return statements
func getOriginalReplicasInt32(r Workload) (originalReplicas int32, originalReplicasSet bool, err error) {
	original, err := getOriginalReplicas(r)
	if err != nil {
		var unsetErr *OriginalReplicasUnsetError
		if errors.As(err, &unsetErr) {
			return 0, false, nil
		}

		return 0, false, fmt.Errorf("failed to get original replicas: %w", err)
	}

	originalInt32, err := original.AsInt32()
	if err != nil {
		return 0, false, fmt.Errorf("failed to convert original replicas to int32: %w", err)
	}

	return originalInt32, true, nil
}
