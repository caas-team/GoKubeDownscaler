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

// ScaleUp scales up the underlying replicaScaledResource.
func (r *replicaScaledWorkload) ScaleUp() (bool, error) {
	originalReplicas, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetErr); ok {
			slog.Debug("original replicas is not set, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
			return false, nil
		}

		return false, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalReplicasInt32, err := originalReplicas.AsInt32()
	if err != nil {
		return false, fmt.Errorf("failed to convert original replicas to int32: %w", err)
	}

	err = r.setReplicas(originalReplicasInt32)
	if err != nil {
		return false, fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(r)

	return true, nil
}

// ScaleDown scales down the underlying replicaScaledResource.
//

func (r *replicaScaledWorkload) ScaleDown(downscaleReplicas values.Replicas) (*metrics.SavedResources, bool, error) {
	downscaleReplicasInt32, err := downscaleReplicas.AsInt32()

	savedResources := metrics.NewSavedResources(0, 0)
	if err != nil {
		return savedResources, false, fmt.Errorf("failed to convert replicas to int32: %w", err)
	}

	currentReplicas, err := r.getReplicas()
	if err != nil {
		return savedResources, false, fmt.Errorf("failed to get current replicas for workload: %w", err)
	}

	currentReplicasInt32, err := currentReplicas.AsInt32()
	if err != nil {
		return savedResources, false, fmt.Errorf("failed to convert current replicas to int32: %w", err)
	}

	if currentReplicasInt32 == downscaleReplicasInt32 {
		var originalReplicasInt32 int32
		var isOriginalReplicasSet bool

		originalReplicasInt32, isOriginalReplicasSet, err = getOriginalReplicasInt32(r)
		if err != nil {
			return savedResources, false, err
		}

		if !isOriginalReplicasSet {
			slog.Debug("workload is already at target scale down replicas, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
			return savedResources, false, nil
		}

		savedResources = r.getSavedResourcesRequests(originalReplicasInt32 - downscaleReplicasInt32)

		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())

		return savedResources, false, nil
	}

	err = r.setReplicas(downscaleReplicasInt32)
	if err != nil {
		return savedResources, false, fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	savedResources = r.getSavedResourcesRequests(currentReplicasInt32 - downscaleReplicasInt32)

	setOriginalReplicas(currentReplicas, r)

	return savedResources, true, nil
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
