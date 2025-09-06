// nolint:ireturn // required for interface-based workflow
package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// replicaScaledWorkload is a wrapper for all resources which are scaled by setting the replica count.
type replicaScaledWorkload struct {
	replicaScaledResource
}

func (r *replicaScaledWorkload) Copy() (Workload, error) {
	if r.replicaScaledResource == nil {
		return nil, newNilUnderlyingObjectError("replicaScaledResource")
	}

	workloadCopy, err := r.replicaScaledResource.Copy()
	if err != nil {
		return nil, newFailedToCompareWorkloadsError("failed to copy suspendScaledResource: %w", err)
	}

	return workloadCopy, nil
}

func (r *replicaScaledWorkload) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	if r.replicaScaledResource == nil {
		return nil, newNilUnderlyingObjectError("replicaScaledResource")
	}

	diff, err := r.replicaScaledResource.Compare(workloadCopy)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError("failed to compare replicaScaledResource: %w", err)
	}

	return diff, nil
}

// ScaleUp scales up the underlying replicaScaledResource.
// nolint: err113 // dynamic errors
func (r *replicaScaledWorkload) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(r)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetErr); ok {
			slog.Debug("original replicas is not set, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalReplicasInt32, err := originalReplicas.AsInt32()
	if err != nil {
		return fmt.Errorf("failed to convert original replicas to int32: %w", err)
	}

	err = r.setReplicas(originalReplicasInt32)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(r)

	return nil
}

// ScaleDown scales down the underlying replicaScaledResource.
// nolint: err113 // dynamic errors
func (r *replicaScaledWorkload) ScaleDown(downscaleReplicas values.Replicas) error {
	downscaleReplicasInt32, err := downscaleReplicas.AsInt32()
	if err != nil {
		return fmt.Errorf("failed to convert replicas to int32: %w", err)
	}

	originalReplicas, err := r.getReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalReplicasInt32, err := originalReplicas.AsInt32()
	if err != nil {
		return fmt.Errorf("failed to convert original replicas to int32: %w", err)
	}

	if originalReplicasInt32 == downscaleReplicasInt32 {
		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
		return nil
	}

	err = r.setReplicas(downscaleReplicasInt32)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	setOriginalReplicas(originalReplicas, r)

	return nil
}
