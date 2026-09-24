//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	appsv1 "k8s.io/api/apps/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// eckPauseAnnotation is the ECK operator's pause-orchestration switch. Unlike a
// first-class suspend, pausing only stops spec-driven reconciliation — the
// operator leaves the node pods and StatefulSets running and keeps housekeeping
// (certs, services, health) alive. So parking an Elasticsearch is two moves:
// set the annotation to "true", THEN scale its node StatefulSets to 0 (its
// children). On wake the annotation is set back to "false" and ECK re-asserts
// the desired node counts through normal reconciliation.
//
// The webhook accepts exactly "true"/"false" — no other spelling.
const (
	eckPauseAnnotation = "eck.k8s.elastic.co/pause-orchestration"
	eckPauseOn         = "true"
	eckPauseOff        = "false"
	// eckClusterNameLabel links a node-set StatefulSet to its parent Elasticsearch.
	eckClusterNameLabel = "elasticsearch.k8s.elastic.co/cluster-name"
)

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var elasticsearchGVK = schema.GroupVersionKind{
	Group: elasticsearchGroup, Version: elasticsearchVersion, Kind: elasticsearchKind,
}

// elasticsearch wraps an unstructured Elasticsearch CR from the ECK operator.
// The unstructured approach avoids importing the operator's API module.
//
// It is suspend-shaped on the pause-orchestration annotation, and it owns
// children: the per-nodeSet StatefulSets, which must be scaled to 0 on park
// because pausing does not stop them.
type elasticsearch struct {
	*unstructured.Unstructured
}

// getElasticsearches is the getResourceFunc for Elasticsearch.
func getElasticsearches(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   elasticsearchGVK.Group,
		Version: elasticsearchGVK.Version,
		Kind:    elasticsearchGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("eck operator CRD not found in cluster, skipping", "kind", elasticsearchGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get elasticsearches: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, elasticsearchGVK)
		results = append(results, &suspendScaledWorkload{&elasticsearch{&item}})
	}

	return results, nil
}

// parseElasticsearchFromBytes parses the admission review and returns the elasticsearch wrapped in a Workload.
func parseElasticsearchFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode elasticsearch: %w", err)
	}

	return &suspendScaledWorkload{&elasticsearch{&u}}, nil
}

// getSuspend gets the current pause state of the cluster and the target downscale state for it.
//
//nolint:nonamedreturns // named returns document the (current, target) contract of the interface
func (e *elasticsearch) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	current := e.GetAnnotations()[eckPauseAnnotation] == eckPauseOn

	currentValue = values.BooleanReplicas(current)
	targetDownscaleState = values.BooleanReplicas(true)

	return currentValue, targetDownscaleState
}

// setSuspend sets the pause-orchestration annotation on the cluster.
// true pauses (park), false resumes (wake).
func (e *elasticsearch) setSuspend(suspend bool) {
	value := eckPauseOff
	if suspend {
		value = eckPauseOn
	}

	annotations := e.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[eckPauseAnnotation] = value
	e.SetAnnotations(annotations)
}

// GetChildren returns the node-set StatefulSets of the Elasticsearch cluster,
// wrapped as replica-scaled workloads so the downscaler scales them to 0 on park.
// They are matched by the eck cluster-name label. This is required because
// pausing orchestration does not stop the running pods.
func (e *elasticsearch) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	list, err := clientsets.Kubernetes.AppsV1().StatefulSets(e.GetNamespace()).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", eckClusterNameLabel, e.GetName()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets for elasticsearch %s/%s: %w", e.GetNamespace(), e.GetName(), err)
	}

	var mutex sync.Mutex

	results := make([]Workload, 0, len(list.Items))

	for i := range list.Items {
		sts := &list.Items[i]
		setGroupVersionKindIfEmpty(sts, appsv1.SchemeGroupVersion.WithKind(statefulSetKind))

		mutex.Lock()

		results = append(results, &replicaScaledWorkload{&statefulSet{sts}})
		mutex.Unlock()
	}

	return results, nil
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
//
// Elasticsearch resources live per-nodeSet under spec.nodeSets[].podTemplate; a
// missing or unparseable value counts as zero so accounting never breaks a scale.
func (e *elasticsearch) getSavedResourcesRequests() *metrics.SavedResources {
	var totalCPU, totalMemory float64

	nodeSets, _, _ := unstructured.NestedSlice(e.Object, "spec", "nodeSets")
	for _, ns := range nodeSets {
		nodeSet, ok := ns.(map[string]any)
		if !ok {
			continue
		}

		count, _, _ := unstructured.NestedInt64(nodeSet, "count")
		if count <= 0 {
			count = 1
		}

		containers, _, _ := unstructured.NestedSlice(nodeSet, "podTemplate", "spec", "containers")
		for _, c := range containers {
			container, ok := c.(map[string]any)
			if !ok {
				continue
			}

			cpu, _, _ := unstructured.NestedString(container, "resources", "requests", "cpu")
			memory, _, _ := unstructured.NestedString(container, "resources", "requests", "memory")
			totalCPU += parseQuantityAsFloat(&cpu) * float64(count)
			totalMemory += parseQuantityAsFloat(&memory) * float64(count)
		}
	}

	return metrics.NewSavedResources(totalCPU, totalMemory)
}

// Copy creates a deep copy of the workload.
func (e *elasticsearch) Copy() (Workload, error) {
	if e.Object == nil {
		return nil, newNilUnderlyingObjectError(e.GetKind())
	}

	return &suspendScaledWorkload{
		suspendScaledResource: &elasticsearch{
			Unstructured: e.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (e *elasticsearch) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	eCopy, ok := sswCopy.suspendScaledResource.(*elasticsearch)
	if !ok {
		return nil, newExpectTypeGotTypeError((*elasticsearch)(nil), sswCopy.suspendScaledResource)
	}

	if e.Object == nil || eCopy.Object == nil {
		return nil, newNilUnderlyingObjectError(e.GetKind())
	}

	diff, err := jsondiff.Compare(e.Object, eCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare %s: %w", e.GetKind(), err)
	}

	return diff, nil
}

// Reget regets the workload to ensure the latest state.
func (e *elasticsearch) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, elasticsearchGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: e.GetNamespace(), Name: e.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", e.GetKind(), e.GetNamespace(), e.GetName(), err)
	}

	e.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (e *elasticsearch) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, e.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", e.GetKind(), e.GetNamespace(), e.GetName(), err)
	}

	return nil
}
