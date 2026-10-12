//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kruisev1alpha1 "github.com/openkruise/kruise/apis/apps/v1alpha1"
	kruisev1beta1 "github.com/openkruise/kruise/apis/apps/v1beta1"
	"github.com/wI2L/jsondiff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// getStatefulSets is the getResourceFunc for KruiseStatefulSets.
func getAdvancedStatefulSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	kruisestatefulset, err := clientsets.Kruise.AppsV1beta1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get advancedstatefulsets: %w", err)
	}

	results := make([]Workload, 0, len(kruisestatefulset.Items))
	for i := range kruisestatefulset.Items {
		setGroupVersionKindIfEmpty(&kruisestatefulset.Items[i], kruisev1beta1.SchemeGroupVersion.WithKind("AdvancedStatefulSet"))
		results = append(results, &replicaScaledWorkload{&advancedStatefulSet{&kruisestatefulset.Items[i]}})
	}

	return results, nil
}

// parseStatefulSetFromBytes parses the admission review and returns the statefulset.
func parseKruiseStatefulSetFromBytes(rawObject []byte) (Workload, error) {
	var meta struct {
		APIVersion string `json:"apiVersion"`
	}
	if err := json.Unmarshal(rawObject, &meta); err != nil {
		return nil, fmt.Errorf("failed to detect apiVersion: %w", err)
	}

	switch meta.APIVersion {
	case "apps.kruise.io/v1alpha1":
		var sts kruisev1alpha1.StatefulSet
		if err := json.Unmarshal(rawObject, &sts); err != nil {
			return nil, fmt.Errorf("failed to decode v1alpha1 AdvancedStatefulSet: %w", err)
		}

		return &replicaScaledWorkload{&advancedStatefulSetV1alpha1{&sts}}, nil
	case "apps.kruise.io/v1beta1":
		var sts kruisev1beta1.StatefulSet
		if err := json.Unmarshal(rawObject, &sts); err != nil {
			return nil, fmt.Errorf("failed to decode v1beta1 AdvancedStatefulSet: %w", err)
		}

		return &replicaScaledWorkload{&advancedStatefulSet{&sts}}, nil
	default:
		return nil, newUnsupportedAPIVersionError(meta.APIVersion)
	}
}

// advancedStatefulSet is a wrapper for StatefulSet.v1beta1.apps.kruise.io to implement the replicaScaledResource interface.
type advancedStatefulSet struct {
	*kruisev1beta1.StatefulSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *advancedStatefulSet) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (s *advancedStatefulSet) getReplicas() (values.Replicas, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(s.Kind, s.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (s *advancedStatefulSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	s.StatefulSet, err = clientsets.Kruise.AppsV1beta1().StatefulSets(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get advancedstatefulset: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the StatefulSet.
func (s *advancedStatefulSet) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	return savedResourcesFromContainers(s.Spec.Template.Spec.Containers, diffReplicas)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *advancedStatefulSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1beta1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update advancedstatefulset: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a statefulSet.
func (s *advancedStatefulSet) Copy() (Workload, error) {
	if s.StatefulSet == nil {
		return nil, newNilUnderlyingObjectError(s.Kind)
	}

	copied := s.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &advancedStatefulSet{
			StatefulSet: copied,
		},
	}, nil
}

// GenerateJSONDiff compares two statefulSet resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (s *advancedStatefulSet) GenerateJSONDiff(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	stsCopy, ok := rswCopy.replicaScaledResource.(*advancedStatefulSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*advancedStatefulSet)(nil), rswCopy.replicaScaledResource)
	}

	if s.StatefulSet == nil || stsCopy.StatefulSet == nil {
		return nil, newNilUnderlyingObjectError(s.Kind)
	}

	diff, err := jsondiff.Compare(s.StatefulSet, stsCopy.StatefulSet)
	if err != nil {
		return nil, fmt.Errorf("failed to compare advancedstatefulsets: %w", err)
	}

	return diff, nil
}

// GeneratePatchDiff compares the workload with another workload and returns the JSON patch bytes.
func (s *advancedStatefulSet) GeneratePatchDiff(workloadCopy Workload) ([]byte, error) {
	patch, err := s.GenerateJSONDiff(workloadCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate advancedStatefulSet patch diff: %w", err)
	}

	patchedDiff, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %w", err)
	}

	return patchedDiff, nil
}

// advancedStatefulSetV1alpha1 is a wrapper for StatefulSet.v1alpha1.apps.kruise.io to implement the replicaScaledResource interface.
type advancedStatefulSetV1alpha1 struct {
	*kruisev1alpha1.StatefulSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *advancedStatefulSetV1alpha1) setReplicas(replicas int32) error {
	s.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (s *advancedStatefulSetV1alpha1) getReplicas() (values.Replicas, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(s.Kind, s.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (s *advancedStatefulSetV1alpha1) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	s.StatefulSet, err = clientsets.Kruise.AppsV1alpha1().StatefulSets(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get advancedstatefulset v1alpha1: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the StatefulSet.
func (s *advancedStatefulSetV1alpha1) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	return savedResourcesFromContainers(s.Spec.Template.Spec.Containers, diffReplicas)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *advancedStatefulSetV1alpha1) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1alpha1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update advancedstatefulset v1alpha1: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a statefulSet.
func (s *advancedStatefulSetV1alpha1) Copy() (Workload, error) {
	if s.StatefulSet == nil {
		return nil, newNilUnderlyingObjectError(s.Kind)
	}

	copied := s.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &advancedStatefulSetV1alpha1{StatefulSet: copied},
	}, nil
}

// GenerateJSONDiff compares two statefulSet resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (s *advancedStatefulSetV1alpha1) GenerateJSONDiff(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	stsCopy, ok := rswCopy.replicaScaledResource.(*advancedStatefulSetV1alpha1)
	if !ok {
		return nil, newExpectTypeGotTypeError((*advancedStatefulSetV1alpha1)(nil), rswCopy.replicaScaledResource)
	}

	if s.StatefulSet == nil || stsCopy.StatefulSet == nil {
		return nil, newNilUnderlyingObjectError(s.Kind)
	}

	diff, err := jsondiff.Compare(s.StatefulSet, stsCopy.StatefulSet)
	if err != nil {
		return nil, fmt.Errorf("failed to compare advancedstatefulsets v1alpha1: %w", err)
	}

	return diff, nil
}

// GeneratePatchDiff compares the workload with another workload and returns the JSON patch bytes.
func (s *advancedStatefulSetV1alpha1) GeneratePatchDiff(workloadCopy Workload) ([]byte, error) {
	patch, err := s.GenerateJSONDiff(workloadCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate advancedStatefulSetV1alpha1 patch diff: %w", err)
	}

	patchedDiff, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %w", err)
	}

	return patchedDiff, nil
}

// savedResourcesFromContainers is a helper function to calculate the total saved resources requests when downscaling based
// on the given containers and diffReplicas.
func savedResourcesFromContainers(containers []corev1.Container, diffReplicas int32) *metrics.SavedResources {
	var totalCPU, totalMemory float64

	for i := range containers {
		c := &containers[i]
		if c.Resources.Requests != nil {
			totalCPU += c.Resources.Requests.Cpu().AsApproximateFloat64()
			totalMemory += c.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	return metrics.NewSavedResources(totalCPU*float64(diffReplicas), totalMemory*float64(diffReplicas))
}

func (s *advancedStatefulSet) Patch(
	clientsets *Clientsets,
	patchType types.PatchType,
	patchData []byte,
	manageFields bool,
	ctx context.Context,
) error {
	if s.StatefulSet == nil {
		return newNilUnderlyingObjectError(s.Kind)
	}

	options := patchOptions(manageFields)

	_, err := clientsets.Kruise.AppsV1beta1().StatefulSets(s.Namespace).Patch(
		ctx,
		s.Name,
		patchType,
		patchData,
		options,
	)
	if err != nil {
		return fmt.Errorf("failed to patch advancedstatefulset: %w", err)
	}

	return nil
}

func (s *advancedStatefulSetV1alpha1) Patch(
	clientsets *Clientsets,
	patchType types.PatchType,
	patchData []byte,
	manageFields bool,
	ctx context.Context,
) error {
	if s.StatefulSet == nil {
		return newNilUnderlyingObjectError(s.Kind)
	}

	options := patchOptions(manageFields)

	_, err := clientsets.Kruise.AppsV1alpha1().StatefulSets(s.Namespace).Patch(
		ctx,
		s.Name,
		patchType,
		patchData,
		options,
	)
	if err != nil {
		return fmt.Errorf("failed to patch advancedstatefulsetv1alpha1: %w", err)
	}

	return nil
}
