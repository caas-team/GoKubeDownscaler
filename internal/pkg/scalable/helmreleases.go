//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/wI2L/jsondiff"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// gethelmReleases is the getResourceFunc for helmReleases.
func getHelmReleases(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var releases helmv2.HelmReleaseList

	err := clientsets.Client.List(ctx, &releases, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get helmreleases: %w", err)
	}

	results := make([]Workload, 0, len(releases.Items))
	for i := range releases.Items {
		setGroupVersionKindIfEmpty(&releases.Items[i], helmv2.GroupVersion.WithKind(helmReleaseKind))
		results = append(results, &suspendScaledWorkload{&helmRelease{&releases.Items[i]}})
	}

	return results, nil
}

// parseHelmReleaseFromBytes parses the admission review and returns the helmrelease wrapped in a Workload.
func parseHelmReleaseFromBytes(rawObject []byte) (Workload, error) {
	var hr helmv2.HelmRelease
	if err := json.Unmarshal(rawObject, &hr); err != nil {
		return nil, fmt.Errorf("failed to decode helmrelease: %w", err)
	}

	return &suspendScaledWorkload{&helmRelease{&hr}}, nil
}

// helmRelease is a wrapper for helmv2.HelmRelease to implement the suspendScaledResource interface.
type helmRelease struct {
	*helmv2.HelmRelease
}

func (h *helmRelease) Reget(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: h.Namespace, Name: h.Name}, h.HelmRelease)
	if err != nil {
		return fmt.Errorf("failed to get helmrelease: %w", err)
	}

	setGroupVersionKindIfEmpty(h.HelmRelease, helmv2.GroupVersion.WithKind(helmReleaseKind))

	return nil
}

func (h *helmRelease) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, h.HelmRelease)
	if err != nil {
		return fmt.Errorf("failed to update helmrelease: %w", err)
	}

	return nil
}

// setSuspend sets the suspend state on the resource.
func (h *helmRelease) setSuspend(suspend bool) {
	h.Spec.Suspend = suspend
}

// getSuspend gets the current suspend state of the resource.
//
//nolint:nonamedreturns // necessary to clarify the return values for this function
func (h *helmRelease) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	current := h.Spec.Suspend

	currentValue = values.BooleanReplicas(current)
	targetDownscaleState = values.BooleanReplicas(true)

	return currentValue, targetDownscaleState
}

// getSavedResourcesRequests calculates the saved resource requests based on the difference in replicas.
func (h *helmRelease) getSavedResourcesRequests() *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// Copy creates a deep copy of the workload.
func (h *helmRelease) Copy() (Workload, error) {
	if h.HelmRelease == nil {
		return nil, newNilUnderlyingObjectError(h.Kind)
	}

	copied := h.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &helmRelease{
			HelmRelease: copied,
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //short names are ok for the workflow of this function
func (h *helmRelease) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	depCopy, ok := sswCopy.suspendScaledResource.(*helmRelease)
	if !ok {
		return nil, newExpectTypeGotTypeError((*helmRelease)(nil), sswCopy.suspendScaledResource)
	}

	if h.HelmRelease == nil || depCopy.HelmRelease == nil {
		return nil, newNilUnderlyingObjectError(h.Kind)
	}

	diff, err := jsondiff.Compare(h.HelmRelease, depCopy.HelmRelease)
	if err != nil {
		return nil, fmt.Errorf("failed to compare helmRelease: %w", err)
	}

	return diff, nil
}
