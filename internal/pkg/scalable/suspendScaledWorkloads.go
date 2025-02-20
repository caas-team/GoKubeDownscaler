package scalable

import (
	"context"
)

// suspendScaledResource provides all the functions needed to scale a resource which is scaled by setting a suspend field.
type suspendScaledResource interface {
	scalableResource
	// Update updates the resource with all changes made to it. It should only be called once on a resource
	Update(clientsets *Clientsets, ctx context.Context) error
	// setSuspend sets the value of the suspend field on the workload
	setSuspend(suspend bool)
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
func (r *suspendScaledWorkload) ScaleDown(_ int32) error {
	r.setSuspend(true)
	return nil
}
