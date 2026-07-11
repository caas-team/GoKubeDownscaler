//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kruisev1beta1 "github.com/openkruise/kruise/apis/apps/v1beta1"
	"github.com/wI2L/jsondiff"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getAdvancedCronJobs is the getResourceFunc for AdvancedCronJobs.
func getAdvancedCronJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	advancedCronJobs, err := clientsets.Kruise.AppsV1beta1().AdvancedCronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get advancedcronjobs: %w", err)
	}

	results := make([]Workload, 0, len(advancedCronJobs.Items))
	for i := range advancedCronJobs.Items {
		results = append(results, &suspendScaledWorkload{&advancedCronJob{&advancedCronJobs.Items[i]}})
	}

	return results, nil
}

// parseAdvancedCronJobFromBytes parses the admission review and returns the advanced cronjob wrapped in a Workload.
func parseAdvancedCronJobFromBytes(rawObject []byte) (Workload, error) {
	var acj kruisev1beta1.AdvancedCronJob
	if err := json.Unmarshal(rawObject, &acj); err != nil {
		return nil, fmt.Errorf("failed to decode advancedcronjob: %w", err)
	}

	return &suspendScaledWorkload{&advancedCronJob{&acj}}, nil
}

// advancedCronJob is a wrapper for advancedcronjob.v1alpha1.apps.kruise.io to implement the suspendScaledResource interface.
type advancedCronJob struct {
	*kruisev1beta1.AdvancedCronJob
}

func (c *advancedCronJob) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	activeJobs := c.Status.Active

	var waitGroup sync.WaitGroup
	var mutex sync.Mutex

	errChannel := make(chan error, len(activeJobs))
	results := make([]Workload, 0, len(activeJobs))

	for _, activeJob := range activeJobs {
		waitGroup.Add(1)

		go func(activeJob v1.ObjectReference) {
			defer waitGroup.Done()

			if c.Status.Type == kruisev1beta1.BroadcastJobTemplate {
				singleBroadcastJob, err := clientsets.Kruise.AppsV1beta1().BroadcastJobs(c.Namespace).Get(ctx, activeJob.Name, metav1.GetOptions{})
				if err != nil {
					errChannel <- fmt.Errorf("failed to get broadcastjob %s: %w", activeJob.Name, err)
					return
				}

				mutex.Lock()

				results = append(results, &suspendScaledWorkload{&broadcastJob{singleBroadcastJob}})
				mutex.Unlock()

				return
			}

			singleJob, err := clientsets.Kubernetes.BatchV1().Jobs(c.Namespace).Get(ctx, activeJob.Name, metav1.GetOptions{})
			if err != nil {
				errChannel <- fmt.Errorf("failed to get job %s: %w", activeJob.Name, err)
				return
			}

			mutex.Lock()

			results = append(results, &suspendScaledWorkload{&job{singleJob}})
			mutex.Unlock()
		}(activeJob)
	}

	waitGroup.Wait()
	close(errChannel)

	allErrors := make([]error, 0, len(errChannel))
	for err := range errChannel {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return nil, errors.Join(allErrors...)
	}

	return results, nil
}

// Reget regets the resource from the Kubernetes API.
func (c *advancedCronJob) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	c.AdvancedCronJob, err = clientsets.Kruise.AppsV1beta1().AdvancedCronJobs(c.Namespace).Get(ctx, c.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get advancedcronjob: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the AdvancedCronJob.
func (c *advancedCronJob) getSavedResourcesRequests() *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	if c.Spec.Template.JobTemplate != nil {
		for i := range c.Spec.Template.JobTemplate.Spec.Template.Spec.Containers {
			container := &c.Spec.Template.JobTemplate.Spec.Template.Spec.Containers[i]
			if container.Resources.Requests != nil {
				totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
				totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
			}
		}

		parallelism := derefInt32(c.Spec.Template.JobTemplate.Spec.Parallelism, 1)
		totalSavedCPU *= float64(parallelism)
		totalSavedMemory *= float64(parallelism)

		return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
	}

	if c.Spec.Template.BroadcastJobTemplate != nil {
		for i := range c.Spec.Template.BroadcastJobTemplate.Spec.Template.Spec.Containers {
			container := &c.Spec.Template.BroadcastJobTemplate.Spec.Template.Spec.Containers[i]
			if container.Resources.Requests != nil {
				totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
				totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
			}
		}

		parallelism := broadcastParallelismToInt32(c.Spec.Template.BroadcastJobTemplate.Spec.Parallelism, 1)
		totalSavedCPU *= float64(parallelism)
		totalSavedMemory *= float64(parallelism)
	}

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// nolint: nonamedreturns // getSuspend gets the current value of the paused field and the target downscale state.
func (c *advancedCronJob) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	current := false
	if c.Spec.Paused != nil {
		current = *c.Spec.Paused
	}

	currentValue = values.BooleanReplicas(current)
	targetDownscaleState = values.BooleanReplicas(true)

	return currentValue, targetDownscaleState
}

// setSuspend sets the value of the paused field on the advancedCronJob.
func (c *advancedCronJob) setSuspend(suspend bool) {
	c.Spec.Paused = &suspend
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (c *advancedCronJob) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1beta1().AdvancedCronJobs(c.Namespace).Update(ctx, c.AdvancedCronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update advancedcronjob: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a suspendScaledWorkload wrapping an advancedCronJob.
func (c *advancedCronJob) Copy() (Workload, error) {
	if c.AdvancedCronJob == nil {
		return nil, newNilUnderlyingObjectError(c.Kind)
	}

	copied := c.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &advancedCronJob{AdvancedCronJob: copied},
	}, nil
}

// Compare compares two advancedCronJob resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (c *advancedCronJob) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	acjCopy, ok := sswCopy.suspendScaledResource.(*advancedCronJob)
	if !ok {
		return nil, newExpectTypeGotTypeError((*advancedCronJob)(nil), sswCopy.suspendScaledResource)
	}

	if c.AdvancedCronJob == nil || acjCopy.AdvancedCronJob == nil {
		return nil, newNilUnderlyingObjectError(c.Kind)
	}

	diff, err := jsondiff.Compare(c.AdvancedCronJob, acjCopy.AdvancedCronJob)
	if err != nil {
		return nil, fmt.Errorf("failed to compare advancedcronjobs: %w", err)
	}

	return diff, nil
}
