//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStatefulSets is the getResourceFunc for StatefulSets.
func getStatefulSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	statefulsets, err := clientsets.Kubernetes.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}

	results := make([]Workload, 0, len(statefulsets.Items))
	for i := range statefulsets.Items {
		results = append(results, &replicaScaledWorkload{&statefulSet{&statefulsets.Items[i]}})
	}

	return results, nil
}

// parseStatefulSetFromAdmissionRequest parses the admission review and returns the statefulset.
//
//nolint:ireturn //required for interface-based factory
func parseStatefulSetFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var sts appsv1.StatefulSet
	if err := json.Unmarshal(review.Request.Object.Raw, &sts); err != nil {
		return nil, fmt.Errorf("failed to decode Deployment: %w", err)
	}

	return &replicaScaledWorkload{&statefulSet{&sts}}, nil
}

// deepCopyStatefulSet creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a statefulSet.
//
//nolint:ireturn,varnamelen //required for interface-based workflow
func deepCopyStatefulSet(w Workload) (Workload, error) {
	rsw, ok := w.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), w)
	}

	sts, ok := rsw.replicaScaledResource.(*statefulSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*statefulSet)(nil), rsw.replicaScaledResource)
	}

	if sts.StatefulSet == nil {
		return nil, newNilUnderlyingObjectError(sts.Kind)
	}

	copied := sts.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &statefulSet{
			StatefulSet: copied,
		},
	}, nil
}

// compareStatefulSets compares two statefulSet resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func compareStatefulSets(workload, workloadCopy Workload) (jsondiff.Patch, error) {
	rsw, ok := workload.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workload)
	}

	sts, ok := rsw.replicaScaledResource.(*statefulSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*statefulSet)(nil), rsw.replicaScaledResource)
	}

	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	stsCopy, ok := rswCopy.replicaScaledResource.(*statefulSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*statefulSet)(nil), rswCopy.replicaScaledResource)
	}

	if sts.StatefulSet == nil || stsCopy.StatefulSet == nil {
		return nil, newNilUnderlyingObjectError(sts.Kind)
	}

	diff, err := jsondiff.Compare(sts.StatefulSet, stsCopy.StatefulSet)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError(sts.Kind, err)
	}

	return diff, nil
}

// statefulset is a wrapper for statefulset.v1.apps to implement the replicaScaledResource interface.
type statefulSet struct {
	*appsv1.StatefulSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *statefulSet) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (s *statefulSet) getReplicas() (values.Replicas, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(s.Kind, s.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (s *statefulSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	s.StatefulSet, err = clientsets.Kubernetes.AppsV1().StatefulSets(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *statefulSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}

	return nil
}
