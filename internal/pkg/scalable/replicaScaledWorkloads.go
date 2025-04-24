package scalable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// replicaScaledResource provides all the functions needed to scale a resource which is scaled by setting the replica count.
type replicaScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// setReplicas sets the replicas of the workload
	setReplicas(replicas int32) error
	// getReplicas gets the replicas of the workload
	getReplicas() (int32, error)
}

// replicaScaledWorkload is a wrapper for all resources which are scaled by setting the replica count.
type replicaScaledWorkload struct {
	replicaScaledResource
}

// ScaleUp scales up the underlying replicaScaledResource.
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

	err = r.setReplicas(*originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(r)

	return nil
}

// ScaleDown scales down the underlying replicaScaledResource.
func (r *replicaScaledWorkload) ScaleDown(downscaleReplicas int32) error {
	originalReplicas, err := r.getReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already scaled down, skipping", "workload", r.GetName(), "namespace", r.GetNamespace())
		return nil
	}

	err = r.setReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	setOriginalReplicas(originalReplicas, r)

	return nil
}
