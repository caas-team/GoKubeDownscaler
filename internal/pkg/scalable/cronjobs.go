package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	admissionv1 "k8s.io/api/admission/v1"
	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCronJobs is the getResourceFunc for CronJobs.
func getCronJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	cronjobs, err := clientsets.Kubernetes.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cronjobs: %w", err)
	}

	results := make([]Workload, 0, len(cronjobs.Items))
	for i := range cronjobs.Items {
		results = append(results, &suspendScaledWorkload{&cronJob{&cronjobs.Items[i]}})
	}

	return results, nil
}

// parseCronJobFromAdmissionRequest parses the admission review and returns the cronjob wrapped in a Workload.
//
//nolint:ireturn //required for interface-based factory
func parseCronJobFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var cj batch.CronJob
	if err := json.Unmarshal(review.Request.Object.Raw, &cj); err != nil {
		return nil, fmt.Errorf("failed to decode cronjob: %w", err)
	}

	return &suspendScaledWorkload{&cronJob{&cj}}, nil
}

// deepCopyCronJob creates a deep copy of the given Workload, which is expected to be a suspendScaledWorkload wrapping a cronJob.
//
//nolint:ireturn,varnamelen //required for interface-based workflow
func deepCopyCronJob(w Workload) (Workload, error) {
	ssw, ok := w.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), w)
	}

	cj, ok := ssw.suspendScaledResource.(*cronJob)
	if !ok {
		return nil, newExpectTypeGotTypeError((*cronJob)(nil), ssw.suspendScaledResource)
	}

	if cj.CronJob == nil {
		return nil, newNilUnderlyingObjectError("cronJob not found")
	}

	copied := cj.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &cronJob{
			CronJob: copied,
		},
	}, nil
}

// cronJob is a wrapper for cronjob.v1.batch to implement the suspendScaledResource interface.
type cronJob struct {
	*batch.CronJob
}

func (c *cronJob) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	activeJobs := c.Status.Active

	var waitGroup sync.WaitGroup

	var mutex sync.Mutex

	errChannel := make(chan error, len(activeJobs))
	results := make([]Workload, 0, len(activeJobs))

	for _, activeJob := range activeJobs {
		waitGroup.Add(1)

		go func(activeJob v1.ObjectReference) {
			defer waitGroup.Done()

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
func (c *cronJob) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	c.CronJob, err = clientsets.Kubernetes.BatchV1().CronJobs(c.Namespace).Get(ctx, c.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cronjob: %w", err)
	}

	return nil
}

// setSuspend sets the value of the suspend field on the cronJob.
func (c *cronJob) setSuspend(suspend bool) {
	c.Spec.Suspend = &suspend
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (c *cronJob) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.BatchV1().CronJobs(c.Namespace).Update(ctx, c.CronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cronjob: %w", err)
	}

	return nil
}
