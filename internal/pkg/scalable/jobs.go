package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getDeployments is the getResourceFunc for Jobs.
func getJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
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

// parseJobFromAdmissionRequest parses the admission review and returns the job wrapped in a Workload.
//
//nolint:ireturn,varnamelen //required for interface-based workflow
func deepCopyJob(w Workload) (Workload, error) {
	ssw, ok := w.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), w)
	}

	jb, ok := ssw.suspendScaledResource.(*job)
	if !ok {
		return nil, newExpectTypeGotTypeError((*job)(nil), ssw.suspendScaledResource)
	}

	if jb.Job == nil {
		return nil, newNilUnderlyingObjectError("job not found")
	}

	copied := jb.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &job{
			Job: copied,
		},
	}, nil
}

// parseCronJobFromAdmissionRequest parses the admission review and returns the cronjob.
//
//nolint:ireturn //required for interface-based factory
func parseJobFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var j batch.Job
	if err := json.Unmarshal(review.Request.Object.Raw, &j); err != nil {
		return nil, fmt.Errorf("failed to decode job: %w", err)
	}

	return &suspendScaledWorkload{&job{&j}}, nil
}

// job is a wrapper for job.v1.batch to implement the suspendScaledResource interface.
type job struct {
	*batch.Job
}

// setSuspend sets the value of the suspend field on the job.
func (j *job) setSuspend(suspend bool) {
	j.Spec.Suspend = &suspend
}

// Reget regets the resource from the Kubernetes API.
func (j *job) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	j.Job, err = clientsets.Kubernetes.BatchV1().Jobs(j.Namespace).Get(ctx, j.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the Job.
//

func (j *job) getSavedResourcesRequests() *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range j.Spec.Template.Spec.Containers {
		container := &j.Spec.Template.Spec.Containers[i] // use pointer to avoid copy
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	totalSavedCPU *= float64(*j.Spec.Parallelism)
	totalSavedMemory *= float64(*j.Spec.Parallelism)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (j *job) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.BatchV1().Jobs(j.Namespace).Update(ctx, j.Job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}
