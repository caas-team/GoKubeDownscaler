//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kruisev1beta1 "github.com/openkruise/kruise/apis/apps/v1beta1"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getBroadcastJobs is the getResourceFunc for BroadcastJobs.
func getBroadcastJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	broadcastJobs, err := clientsets.Kruise.AppsV1beta1().BroadcastJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get broadcastjobs: %w", err)
	}

	results := make([]Workload, 0, len(broadcastJobs.Items))
	for i := range broadcastJobs.Items {
		results = append(results, &suspendScaledWorkload{&broadcastJob{&broadcastJobs.Items[i]}})
	}

	return results, nil
}

// parseBroadcastJobFromBytes parses the admission review and returns the broadcastjob.
func parseBroadcastJobFromBytes(rawObject []byte) (Workload, error) {
	var bj kruisev1beta1.BroadcastJob
	if err := json.Unmarshal(rawObject, &bj); err != nil {
		return nil, fmt.Errorf("failed to decode broadcastjob: %w", err)
	}

	return &suspendScaledWorkload{&broadcastJob{&bj}}, nil
}

// broadcastJob is a wrapper for broadcastjob.v1alpha1.apps.kruise.io to implement the suspendScaledResource interface.
type broadcastJob struct {
	*kruisev1beta1.BroadcastJob
}

// nolint: nonamedreturns // getSuspend gets the current value of the paused field and the target downscale state.
func (b *broadcastJob) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	currentValue = values.BooleanReplicas(b.Spec.Paused)
	targetDownscaleState = values.BooleanReplicas(true)

	return currentValue, targetDownscaleState
}

// setSuspend sets the value of the paused field on the BroadcastJob.
func (b *broadcastJob) setSuspend(suspend bool) {
	b.Spec.Paused = suspend
}

// Reget regets the resource from the Kubernetes API.
func (b *broadcastJob) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	b.BroadcastJob, err = clientsets.Kruise.AppsV1beta1().BroadcastJobs(b.Namespace).Get(ctx, b.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get broadcastjob: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the BroadcastJob.
func (b *broadcastJob) getSavedResourcesRequests() *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range b.Spec.Template.Spec.Containers {
		container := &b.Spec.Template.Spec.Containers[i]
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	parallelism := broadcastParallelismToInt32(b.Spec.Parallelism, b.Status.Desired)
	totalSavedCPU *= float64(parallelism)
	totalSavedMemory *= float64(parallelism)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (b *broadcastJob) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1beta1().BroadcastJobs(b.Namespace).Update(ctx, b.BroadcastJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update broadcastjob: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a suspendScaledWorkload wrapping a broadcastJob.
func (b *broadcastJob) Copy() (Workload, error) {
	if b.BroadcastJob == nil {
		return nil, newNilUnderlyingObjectError(b.Kind)
	}

	copied := b.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &broadcastJob{BroadcastJob: copied},
	}, nil
}

// Compare compares two broadcastJob resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (b *broadcastJob) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	bjCopy, ok := sswCopy.suspendScaledResource.(*broadcastJob)
	if !ok {
		return nil, newExpectTypeGotTypeError((*broadcastJob)(nil), sswCopy.suspendScaledResource)
	}

	if b.BroadcastJob == nil || bjCopy.BroadcastJob == nil {
		return nil, newNilUnderlyingObjectError(b.Kind)
	}

	diff, err := jsondiff.Compare(b.BroadcastJob, bjCopy.BroadcastJob)
	if err != nil {
		return nil, fmt.Errorf("failed to compare broadcastjobs: %w", err)
	}

	return diff, nil
}
