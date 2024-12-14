package scalable

import (
	"context"
	"fmt"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getJobs is the getResourceFunc for Jobs
func getJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	jobs, err := clientsets.Kubernetes.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	for _, item := range jobs.Items {
		results = append(results, &suspendScaledWorkload{&job{&item}})
	}
	return results, nil
}

// getJob is the getResourcesFunc for a single CronJob
func getJob(name string, namespace string, clientsets *Clientsets, ctx context.Context) (Workload, error) {
	var result Workload
	batchJob, err := clientsets.Kubernetes.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get job: %w", err)
	}
	result = &suspendScaledWorkload{&job{batchJob}}
	return result, nil
}

// job is a wrapper for batch/v1.Job to implement the suspendScaledResource interface
type job struct {
	*batch.Job
}

// GetResourceType returns the name of the workload type
func (j *job) GetResourceType() string {
	return "job"
}

// setSuspend sets the value of the suspend field on the job
func (j *job) setSuspend(suspend bool) {
	j.Spec.Suspend = &suspend
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (j *job) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.BatchV1().Jobs(j.Namespace).Update(ctx, j.Job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}
