package scalable

import (
	"context"
	"fmt"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCronJobs is the getResourceFunc for CronJobs.
func getCronJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	cronjobs, err := clientsets.Kubernetes.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to convert workload to cronjobs", errConversionFailed)
	}

	results := make([]Workload, 0, len(cronjobs.Items))
	for i := range cronjobs.Items {
		results = append(results, &suspendScaledWorkload{&cronJob{&cronjobs.Items[i]}})
	}

	return results, nil
}

// getCronJobsChildren is the getResourceFunc for CronJobs children (Jobs).
func getCronJobsChildren(workload Workload, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	cronjob, ok := workload.(*suspendScaledWorkload).suspendScaledResource.(*cronJob)
	if !ok {
		return nil, fmt.Errorf("%w: %v", errConversionFailed, workload)
	}

	activeJobs := cronjob.Status.Active
	results := make([]Workload, 0, len(activeJobs))

	for _, activeJob := range activeJobs {
		singleJob, err := clientsets.Kubernetes.BatchV1().Jobs(workload.GetNamespace()).Get(ctx, activeJob.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get jobs: %w", err)
		}

		results = append(results, &suspendScaledWorkload{&job{singleJob}})
	}

	return results, nil
}

// cronJob is a wrapper for cronjob.v1.batch to implement the suspendScaledResource interface.
type cronJob struct {
	*batch.CronJob
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
