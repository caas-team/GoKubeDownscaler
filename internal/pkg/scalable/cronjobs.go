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
func getCronJobs(namespace string, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	cronjobs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get cronjobs: %w", err)
	}
	for _, item := range cronjobs.Items {
		results = append(results, CronJob{&item})
	}
	return results, nil
}

// CronJob is a wrapper for batch/v1.CronJob to implement the scalableResource interface
type CronJob struct {
	*batch.CronJob
}

// SetSuspend sets the state of the field spec.Suspend to a new value
func (c CronJob) SetSuspend(suspend bool) {
	c.Spec.Suspend = &suspend
}

// GetSuspend gets the current state of the field spec.Suspend
func (c CronJob) GetSuspend() (bool, error) {
	suspend := c.Spec.Suspend
	if c.Spec.Suspend == nil {
		return false, errNoSuspendSpecified
	}
	return bool(*suspend), nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (c CronJob) Update(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) error {
	_, err := clientset.BatchV1().CronJobs(c.Namespace).Update(ctx, c.CronJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cronjob: %w", err)
	}
	return nil
}
