package scalable

import (
	"context"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

// suspendScaledResource provides all the functions needed to scale a resource which is scaled by setting a suspend field.
type suspendScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// setSuspend sets the value of the suspend field on the workload
	setSuspend(suspend bool)
	// getSavedResourcesRequests returns the saved CPU and memory requests for the workload based on the downscale replicas.
	getSavedResourcesRequests() (float64, float64)
}

// suspendScaledWorkload is a wrapper for all resources which are scaled by setting a suspend field.
type suspendScaledWorkload struct {
	suspendScaledResource
}

// ScaleUp scales up the underlying suspendScaledResource.
func (r *suspendScaledWorkload) ScaleUp() error {
	r.setSuspend(false)
	return nil
}

// ScaleDown scales down the underlying suspendScaledResource.
//
//nolint:nonamedreturns // using named return values for clarity and to simplify return statements
func (r *suspendScaledWorkload) ScaleDown(_ values.Replicas) (totalSavedCPU, totalSavedMemory float64, err error) {
	totalSavedCPU, totalSavedMemory = r.getSavedResourcesRequests()

	r.setSuspend(true)

	return totalSavedCPU, totalSavedMemory, nil
}
