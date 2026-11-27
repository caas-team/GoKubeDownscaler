//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var autoscalingRunnerSetGVK = schema.GroupVersionKind{
	Group: "actions.github.com", Version: "v1alpha1", Kind: autoscalingRunnerSetKind,
}

// getAutoscalingRunnerSets is the getResourceFunc for AutoscalingRunnerSets.
func getAutoscalingRunnerSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   autoscalingRunnerSetGVK.Group,
		Version: autoscalingRunnerSetGVK.Version,
		Kind:    autoscalingRunnerSetGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to get autoscalingrunnersets: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, autoscalingRunnerSetGVK)
		results = append(results, &replicaScaledWorkload{&autoscalingRunnerSet{&item}})
	}

	return results, nil
}

// parseAutoscalingRunnerSetFromBytes parses the admission review and returns the autoscalingrunnerset wrapped in a Workload.
func parseAutoscalingRunnerSetFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode autoscalingrunnerset: %w", err)
	}

	return &replicaScaledWorkload{&autoscalingRunnerSet{&u}}, nil
}

// autoscalingRunnerSet wraps an unstructured AutoscalingRunnerSet. The unstructured approach
// preserves fields unknown to the typed ARC API (e.g. spec.listenerMetrics) when updating.
type autoscalingRunnerSet struct {
	*unstructured.Unstructured
}

func (a *autoscalingRunnerSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, autoscalingRunnerSetGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: a.GetNamespace(), Name: a.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get autoscalingrunnerset: %w", err)
	}

	a.Unstructured = fresh

	return nil
}

func (a *autoscalingRunnerSet) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, a.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update autoscalingrunnerset: %w", err)
	}

	return nil
}

// setReplicas sets the amount of replicas on the resource.
func (a *autoscalingRunnerSet) setReplicas(replicas int32) error {
	if err := unstructured.SetNestedField(a.Object, int64(replicas), "spec", "minRunners"); err != nil {
		return fmt.Errorf("failed to set spec.minRunners for %s %s/%s: %w", a.GetKind(), a.GetNamespace(), a.GetName(), err)
	}

	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (a *autoscalingRunnerSet) getReplicas() (values.Replicas, error) {
	val, found, err := unstructured.NestedFieldNoCopy(a.Object, "spec", "minRunners")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.minRunners for %s %s/%s: %w", a.GetKind(), a.GetNamespace(), a.GetName(), err)
	}

	if !found {
		return nil, newNoReplicasError(a.GetKind(), a.GetName())
	}

	replicas, ok := unstructuredReplicasToInt32(val)
	if !ok {
		return nil, newUnexpectedReplicasTypeError(val, a.GetKind(), a.GetNamespace(), a.GetName())
	}

	return values.AbsoluteReplicas(replicas), nil
}

// getSavedResourcesRequests calculates the saved resource requests based on the difference in replicas.
func (a *autoscalingRunnerSet) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	templateSpec, found, err := unstructured.NestedMap(a.Object, "spec", "template", "spec")
	if err != nil || !found {
		return metrics.NewSavedResources(0, 0)
	}

	var podSpec corev1.PodSpec
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(templateSpec, &podSpec); err != nil {
		return metrics.NewSavedResources(0, 0)
	}

	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	totalSavedCPU *= float64(diffReplicas)
	totalSavedMemory *= float64(diffReplicas)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Copy creates a deep copy of the workload.
// Must use DeepCopy() because Unstructured wraps a map, so a struct copy would be shallow.
func (a *autoscalingRunnerSet) Copy() (Workload, error) {
	if a.Unstructured == nil || a.Object == nil {
		return nil, newNilUnderlyingObjectError(a.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &autoscalingRunnerSet{
			Unstructured: a.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (a *autoscalingRunnerSet) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	depCopy, ok := rswCopy.replicaScaledResource.(*autoscalingRunnerSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*autoscalingRunnerSet)(nil), rswCopy.replicaScaledResource)
	}

	if a.Unstructured == nil || depCopy.Unstructured == nil {
		return nil, newNilUnderlyingObjectError(a.GetKind())
	}

	diff, err := jsondiff.Compare(a.Object, depCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare autoscalingrunnerset: %w", err)
	}

	return diff, nil
}
