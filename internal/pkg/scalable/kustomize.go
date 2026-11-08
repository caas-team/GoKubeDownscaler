//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kustomize "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/wI2L/jsondiff"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// getKustomization is the getResourceFunc for Kustomizations.
func getKustomization(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var kustomizations kustomize.KustomizationList

	err := clientsets.Client.List(ctx, &kustomizations, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get kustomizations: %w", err)
	}

	results := make([]Workload, 0, len(kustomizations.Items))
	for i := range kustomizations.Items {
		setGroupVersionKindIfEmpty(&kustomizations.Items[i], kustomize.GroupVersion.WithKind(kustomizationKind))
		results = append(results, &suspendScaledWorkload{&kustomization{&kustomizations.Items[i]}})
	}

	return results, nil
}

// parseKustomizationFromBytes parses the admission review and returns the kustomization wrapped in a Workload.
func parseKustomizationFromBytes(rawObject []byte) (Workload, error) {
	var k kustomize.Kustomization
	if err := json.Unmarshal(rawObject, &k); err != nil {
		return nil, fmt.Errorf("failed to decode kustomization: %w", err)
	}

	return &suspendScaledWorkload{&kustomization{&k}}, nil
}

// kustomization is a wrapper for kustomize.v1 to implement the suspendScaledResource interface.
type kustomization struct {
	*kustomize.Kustomization
}

// logUpscaleSuccessful logs a successful resume operation or dry-run resume operation.
func (k *kustomization) logUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("unsuspended", "suspend", k, summary, dryRun, logger)
}

// logDownscaleSuccessful logs a successful suspend operation or dry-run suspend operation.
func (k *kustomization) logDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("suspended", "suspend", k, summary, dryRun, logger)
}

func (k *kustomization) Reget(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: k.Namespace, Name: k.Name}, k.Kustomization)
	if err != nil {
		return fmt.Errorf("failed to get kustomization: %w", err)
	}

	setGroupVersionKindIfEmpty(k.Kustomization, kustomize.GroupVersion.WithKind(kustomizationKind))

	return nil
}

func (k *kustomization) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, k.Kustomization)
	if err != nil {
		return fmt.Errorf("failed to update kustomization: %w", err)
	}

	return nil
}

// setSuspend sets the suspend state on the resource.
func (k *kustomization) setSuspend(suspend bool) {
	k.Spec.Suspend = suspend
}

// getSuspend gets the current suspend state of the resource.
//
//nolint:nonamedreturns // necessary to clarify the return values for this function
func (k *kustomization) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	current := k.Spec.Suspend

	currentValue = values.BooleanReplicas(current)
	targetDownscaleState = values.BooleanReplicas(true)

	return currentValue, targetDownscaleState
}

// getSavedResourcesRequests calculates the saved resource requests based on the difference in replicas.
func (k *kustomization) getSavedResourcesRequests() *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// Copy creates a deep copy of the workload.
func (k *kustomization) Copy() (Workload, error) {
	if k.Kustomization == nil {
		return nil, newNilUnderlyingObjectError(k.Kind)
	}

	copied := k.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &kustomization{
			Kustomization: copied,
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //short names are ok for the workflow of this function
func (k *kustomization) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	depCopy, ok := sswCopy.suspendScaledResource.(*kustomization)
	if !ok {
		return nil, newExpectTypeGotTypeError((*kustomization)(nil), sswCopy.suspendScaledResource)
	}

	if k.Kustomization == nil || depCopy.Kustomization == nil {
		return nil, newNilUnderlyingObjectError(k.Kind)
	}

	diff, err := jsondiff.Compare(k.Kustomization, depCopy.Kustomization)
	if err != nil {
		return nil, fmt.Errorf("failed to compare kustomization: %w", err)
	}

	return diff, nil
}
