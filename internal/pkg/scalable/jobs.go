//nolint:dupl // this code is very similar for every resource, but its not really abstractable to avoid more duplication
package scalable

import (
	"context"
	"fmt"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getDeployments is the getResourceFunc for Jobs.
func getJobs(name, namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	if name != "" {
		results := make([]Workload, 0, 1)

		singleJob, err := clientsets.Kubernetes.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get job: %w", err)
		}

		results = append(results, &suspendScaledWorkload{&job{singleJob}})

		return results, nil
	}

	jobs, err := clientsets.Kubernetes.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	results := make([]Workload, 0, len(jobs.Items))
	for i := range jobs.Items {
		results = append(results, &suspendScaledWorkload{&job{&jobs.Items[i]}})
	}

	return results, nil
}

// job is a wrapper for batch/v1.Job to implement the suspendScaledResource interface.
type job struct {
	*batch.Job
}

// setSuspend sets the value of the suspend field on the job.
func (j *job) setSuspend(suspend bool) {
	j.Spec.Suspend = &suspend
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (j *job) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.BatchV1().Jobs(j.Namespace).Update(ctx, j.Job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}
