//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kruisev1beta1 "github.com/openkruise/kruise/apis/apps/v1beta1"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// getImagePullJobs is the getResourceFunc for ImagePullJobs.
func getImagePullJobs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	imagePullJobs, err := clientsets.Kruise.AppsV1beta1().ImagePullJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get imagepulljobs: %w", err)
	}

	results := make([]Workload, 0, len(imagePullJobs.Items))
	for i := range imagePullJobs.Items {
		setGroupVersionKindIfEmpty(&imagePullJobs.Items[i], kruisev1beta1.SchemeGroupVersion.WithKind("ImagePullJob"))
		results = append(results, &replicaScaledWorkload{&imagePullJob{&imagePullJobs.Items[i]}})
	}

	return results, nil
}

// parseImagePullJobFromBytes parses the admission review and returns the ImagePullJob.
func parseImagePullJobFromBytes(rawObject []byte) (Workload, error) {
	var bj kruisev1beta1.ImagePullJob
	if err := json.Unmarshal(rawObject, &bj); err != nil {
		return nil, fmt.Errorf("failed to decode imagepulljob: %w", err)
	}

	return &replicaScaledWorkload{&imagePullJob{&bj}}, nil
}

// imagePullJob is a wrapper for imagepulljob.v1beta1.apps.kruise.io to implement the replicaScaledResource interface.
type imagePullJob struct {
	*kruisev1beta1.ImagePullJob
}

// setReplicas sets the parallelism on the resource. Changes won't be made on Kubernetes until Update() is called.
func (i *imagePullJob) setReplicas(replicas int32) error {
	parallelism := intstr.FromInt32(replicas)
	i.Spec.Parallelism = &parallelism

	return nil
}

// getReplicas gets the current parallelism of the resource.
func (i *imagePullJob) getReplicas() (values.Replicas, error) {
	parallelism := kruiseParallelismToInt32(i.Spec.Parallelism, i.Status.Desired)

	return values.AbsoluteReplicas(parallelism), nil
}

// Reget regets the resource from the Kubernetes API.
func (i *imagePullJob) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	i.ImagePullJob, err = clientsets.Kruise.AppsV1beta1().ImagePullJobs(i.Namespace).Get(ctx, i.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ImagePullJob: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the ImagePullJob.
func (i *imagePullJob) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	_ = diffReplicas

	// ImagePullJob defines image pulling tasks but no container resource requests in spec.
	return metrics.NewSavedResources(0, 0)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (i *imagePullJob) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1beta1().ImagePullJobs(i.Namespace).Update(ctx, i.ImagePullJob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ImagePullJob: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping an ImagePullJob.
func (i *imagePullJob) Copy() (Workload, error) {
	if i.ImagePullJob == nil {
		return nil, newNilUnderlyingObjectError(i.Kind)
	}

	copied := i.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &imagePullJob{ImagePullJob: copied},
	}, nil
}

// Compare compares two ImagePullJob resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (i *imagePullJob) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	bjCopy, ok := rswCopy.replicaScaledResource.(*imagePullJob)
	if !ok {
		return nil, newExpectTypeGotTypeError((*imagePullJob)(nil), rswCopy.replicaScaledResource)
	}

	if i.ImagePullJob == nil || bjCopy.ImagePullJob == nil {
		return nil, newNilUnderlyingObjectError(i.Kind)
	}

	diff, err := jsondiff.Compare(i.ImagePullJob, bjCopy.ImagePullJob)
	if err != nil {
		return nil, fmt.Errorf("failed to compare ImagePullJobs: %w", err)
	}

	return diff, nil
}
