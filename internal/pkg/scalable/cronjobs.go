package scalable

import (
	"context"
	"fmt"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// getDeployments is the getResourceFunc for Deployments
func getCronJobs(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	cronjobs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get cronjobs: %w", err)
	}
	for _, item := range cronjobs.Items {
		results = append(results, &cronJob{&item})
	}
	return results, nil
}

// cronJob is a wrapper for batch/v1.cronJob to implement the Workload interface
type cronJob struct {
	*batch.CronJob
}

// ScaleUp upscale the resource when the downscale period ends
func (c *cronJob) ScaleUp() error {
	newSuspend := false
	c.Spec.Suspend = &newSuspend
	return nil
}

// ScaleDown downscale the resource when the downscale period starts
func (c *cronJob) ScaleDown(_ int) error {
	newSuspend := true
	c.Spec.Suspend = &newSuspend
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (c *cronJob) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.BatchV1().CronJobs(c.Namespace).Update(ctx, c.CronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cronjob: %w", err)
	}
	return nil
}
