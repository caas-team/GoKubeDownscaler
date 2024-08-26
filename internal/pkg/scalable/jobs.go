package scalable

import (
	"context"
	"fmt"
	"log/slog"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// getDeployments is the getResourceFunc for Deployments
func getJobs(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	jobs, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	for _, item := range jobs.Items {
		results = append(results, job{&item})
	}
	return results, nil
}

// job is a wrapper for batch/v1.job to implement the scalableResource interface
type job struct {
	*batch.Job
}

// setSuspend sets the state of the field spec.Suspend to a new value
func (j job) setSuspend(suspend bool) {
	j.Spec.Suspend = &suspend
}

// getSuspend gets the current state of the field spec.Suspend
func (j job) getSuspend() (bool, error) {
	suspend := j.Spec.Suspend
	if j.Spec.Suspend == nil {
		return false, errNoSuspendSpecified
	}
	return bool(*suspend), nil
}

// ScaleUp upscale the resource when the downscale period ends
func (j job) ScaleUp() error {
	const suspend = false

	currentSuspendIsTrue, err := j.getSuspend()
	if err != nil {
		return fmt.Errorf("failed to get current replicas for workload: %w", err)
	}
	if !currentSuspendIsTrue {
		slog.Debug("workload is already upscaled, skipping", "workload", j.GetName(), "namespace", j.GetNamespace())
		return nil
	}

	j.setSuspend(suspend)
	return nil
}

// ScaleDown downscale the resource when the downscale period starts
func (j job) ScaleDown(_ int) error {
	const suspend = true

	currentSuspendIsTrue, err := j.getSuspend()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if currentSuspendIsTrue {
		slog.Debug("workload is already downscaled, skipping", "workload", j.GetName(), "namespace", j.GetNamespace())
		return nil
	}

	j.setSuspend(suspend)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (j job) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.BatchV1().Jobs(j.Namespace).Update(ctx, j.Job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	return nil
}
