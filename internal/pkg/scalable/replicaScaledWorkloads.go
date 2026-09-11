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
)

// replicaScaledResource provides all the functions needed to scale a resource which is scaled by setting the replica count.
type replicaScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
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
func (r *replicaScaledWorkload) LogUpscaleSuccessful(summary ScalingSummary, dryRun bool) {
	logWorkloadScalingMessage("scaled up", "replicas", r, summary, dryRun)
}

// LogDownscaleSuccessful logs a successful downscale using the original workload message style.
func (r *replicaScaledWorkload) LogDownscaleSuccessful(summary ScalingSummary, dryRun bool) {
	logWorkloadScalingMessage("scaled down", "replicas", r, summary, dryRun)
}

// ScaleUp scales up the underlying replicaScaledResource.
func (r *replicaScaledWorkload) ScaleUp() (ScalingSummary, error) {
	var summary ScalingSummary

	currentReplicas, err := r.getReplicas()
	if err != nil {
		return summary, fmt.Errorf("failed to get current replicas for workload: %w", err)
	}

	originalReplicas, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetErr); ok {
			slog.Debug("original replicas is not set, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
			return summary, nil
		}

		return summary, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalReplicasInt32, err := originalReplicas.AsInt32()
	if err != nil {
		return summary, fmt.Errorf("failed to convert original replicas to int32: %w", err)
	}

	err = r.setReplicas(originalReplicasInt32)
	if err != nil {
		return summary, fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(r)

	return ScalingSummary{IsUpdateNeeded: true, FromReplicas: currentReplicas, ToReplicas: originalReplicas}, nil
}

// ScaleDown scales down the underlying replicaScaledResource.
//

func (r *replicaScaledWorkload) ScaleDown(downscaleReplicas values.Replicas) (ScalingSummary, error) {
	downscaleReplicasInt32, err := downscaleReplicas.AsInt32()

	summary := ScalingSummary{SavedResources: metrics.NewSavedResources(0, 0)}
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

	// util.Undefined (-1) is a sentinel for "no current replicas set" (e.g. a ScaledObject without a
	// paused-replicas annotation). It must not be treated as already being at or below the downtime target,
	// otherwise such workloads would never be scaled down.
	if currentReplicasInt32 != util.Undefined && currentReplicasInt32 <= downscaleReplicasInt32 {
		var originalReplicasInt32 int32
		var isOriginalReplicasSet bool

		originalReplicasInt32, isOriginalReplicasSet, err = getOriginalReplicasInt32(r)
		if err != nil {
			return summary, err
		}

		if !isOriginalReplicasSet {
			slog.Debug("workload is at or below target scale down replicas, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

			summary.FromReplicas = currentReplicas
			summary.ToReplicas = downscaleReplicas

			return summary, nil
		}

		summary.SavedResources = r.getSavedResourcesRequests(originalReplicasInt32 - downscaleReplicasInt32)

		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

		summary.FromReplicas = currentReplicas
		summary.ToReplicas = downscaleReplicas

		return summary, nil
	}

	err = r.setReplicas(downscaleReplicasInt32)
	if err != nil {
		return summary, fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	summary.SavedResources = r.getSavedResourcesRequests(currentReplicasInt32 - downscaleReplicasInt32)

	setOriginalReplicas(currentReplicas, r)

	summary.IsUpdateNeeded = true
	summary.FromReplicas = currentReplicas
	summary.ToReplicas = downscaleReplicas

	return summary, nil
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
