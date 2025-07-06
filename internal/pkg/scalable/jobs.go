//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"
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
		return nil, newNilUnderlyingObjectError(jb.Kind)
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

// compareJobs compares two Job resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func compareJobs(workload, workloadCopy Workload) (jsondiff.Patch, error) {
	ssw, ok := workload.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workload)
	}

	j, ok := ssw.suspendScaledResource.(*job)
	if !ok {
		return nil, newExpectTypeGotTypeError((*job)(nil), ssw.suspendScaledResource)
	}

	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	jCopy, ok := sswCopy.suspendScaledResource.(*job)
	if !ok {
		return nil, newExpectTypeGotTypeError((*job)(nil), sswCopy.suspendScaledResource)
	}

	if j.Job == nil || jCopy.Job == nil {
		return nil, newNilUnderlyingObjectError(j.Kind)
	}

	diff, err := jsondiff.Compare(j.Job, jCopy.Job)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError(j.Kind, err)
	}

	return diff, nil
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

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (j *job) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.BatchV1().Jobs(j.Namespace).Update(ctx, j.Job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}
