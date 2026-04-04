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

// valueScaledResource provides all the functions needed to scale a resource which is scaled by setting a suspend field.
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
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// valueScaledWorkload is a wrapper for all resources which are scaled by setting custom value field.
type valueScaledWorkload struct {
	valueScaledResource
}

// ScaleUp scales up the underlying valueScaledResource.
func (v *valueScaledWorkload) ScaleUp() error {
	originalState, err := getOriginalReplicas(v)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			slog.Debug("original replicas is not set, skipping", "workload", v.GetName(), "namespace", v.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	err = v.setValue(originalState)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}

	removeOriginalReplicas(v)

	return nil
}

// ScaleDown scales down the underlying valueScaledResource.
func (v *valueScaledWorkload) ScaleDown(_ values.Replicas) (*metrics.SavedResources, error) {
	currentState, targetScaleDownState, err := v.getValue()
	if err != nil {
		return metrics.NewSavedResources(0, 0), err
	}

	if currentState == targetScaleDownState {
		_, err = getOriginalReplicas(v)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if ok := errors.As(err, &originalReplicasUnsetErr); !ok {
				return metrics.NewSavedResources(0, 0), err
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", v.GetName(), "namespace", v.GetNamespace())

			return metrics.NewSavedResources(0, 0), nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", v.GetName(), "namespace", v.GetNamespace())

		return metrics.NewSavedResources(0, 0), nil
	}

	savedResources := v.getSavedResourcesRequests()

	err = v.setValue(targetScaleDownState)
	if err != nil {
		return metrics.NewSavedResources(0, 0), fmt.Errorf("failed to set replicas for workload: %w", err)
	}

	setOriginalReplicas(currentState, v)

	return savedResources, nil
}
