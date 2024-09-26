package scalable

import (
	"context"
	"fmt"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getDeployments is the getResourceFunc for Jobs
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

// job is a wrapper for batch/v1.Job to implement the suspendScaledResource interface
type job struct {
	*batch.Job
}

// setSuspend sets the value of the suspend field on the job
func (c *job) setSuspend(suspend bool) {
	c.Spec.Suspend = &suspend
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (j *job) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.BatchV1().Jobs(j.Namespace).Update(ctx, j.Job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}
