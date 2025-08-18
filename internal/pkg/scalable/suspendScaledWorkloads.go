// nolint: ireturn // required for interface-based workflow
package scalable

import (
	"context"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
)

// suspendScaledResource provides all the functions needed to scale a resource which is scaled by setting a suspend field.
type suspendScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// setSuspend sets the value of the suspend field on the workload
	setSuspend(suspend bool)
	// Copy creates a deep copy of the workload
	Copy() (Workload, error)
	// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch
	Compare(workloadCopy Workload) (jsondiff.Patch, error)
}

// suspendScaledWorkload is a wrapper for all resources which are scaled by setting a suspend field.
type suspendScaledWorkload struct {
	suspendScaledResource
}

func (r *suspendScaledWorkload) Copy() (Workload, error) {
	if r.suspendScaledResource == nil {
		return nil, newNilUnderlyingObjectError("suspendScaledResource")
	}

	workloadCopy, err := r.suspendScaledResource.Copy()
	if err != nil {
		return nil, newFailedToCompareWorkloadsError("failed to copy suspendScaledResource: %w", err)
	}

	return workloadCopy, nil
}

func (r *suspendScaledWorkload) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	if r.suspendScaledResource == nil {
		return nil, newNilUnderlyingObjectError("suspendScaledResource")
	}

	diff, err := r.suspendScaledResource.Compare(workloadCopy)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError("failed to compare suspendScaledResource: %w", err)
	}

	return diff, nil
}

// ScaleUp scales up the underlying suspendScaledResource.
func (r *suspendScaledWorkload) ScaleUp() error {
	r.setSuspend(false)
	return nil
}

// ScaleDown scales down the underlying suspendScaledResource.
func (r *suspendScaledWorkload) ScaleDown(_ values.Replicas) error {
	r.setSuspend(true)
	return nil
}
