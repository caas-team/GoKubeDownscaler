//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var rabbitmqClusterGVK = schema.GroupVersionKind{
	Group: rabbitmqGroup, Version: rabbitmqVersion, Kind: rabbitmqClusterKind,
}

// rabbitmqCluster wraps an unstructured RabbitmqCluster CR from the RabbitMQ Cluster Operator.
// The unstructured approach is used to avoid importing the operator's API module.
//
// The operator supports spec.replicas: 0 natively. On the way down it records the previous
// count in the rabbitmq.com/before-zero-replicas-configured annotation, and on the way up it
// only accepts that same count. The downscaler's original-replicas annotation carries the same
// value, so restoring from it is always accepted.
type rabbitmqCluster struct {
	*unstructured.Unstructured
}

// getRabbitmqClusters is the getResourceFunc for RabbitmqCluster.
func getRabbitmqClusters(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   rabbitmqClusterGVK.Group,
		Version: rabbitmqClusterGVK.Version,
		Kind:    rabbitmqClusterGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("rabbitmq cluster operator CRD not found in cluster, skipping", "kind", rabbitmqClusterGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get rabbitmqclusters: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, rabbitmqClusterGVK)
		results = append(results, &replicaScaledWorkload{&rabbitmqCluster{&item}})
	}

	return results, nil
}

// parseRabbitmqClusterFromBytes parses the admission review and returns the rabbitmqcluster wrapped in a Workload.
func parseRabbitmqClusterFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode rabbitmqcluster: %w", err)
	}

	return &replicaScaledWorkload{&rabbitmqCluster{&u}}, nil
}

// getReplicas gets the current amount of replicas of the resource.
func (r *rabbitmqCluster) getReplicas() (values.Replicas, error) {
	val, found, err := unstructured.NestedFieldNoCopy(r.Object, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.replicas for %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	if !found {
		return nil, newNoReplicasError(r.GetKind(), r.GetName())
	}

	replicas, ok := unstructuredReplicasToInt32(val)
	if !ok {
		return nil, newUnexpectedReplicasTypeError(val, r.GetKind(), r.GetNamespace(), r.GetName())
	}

	return values.AbsoluteReplicas(replicas), nil
}

// setReplicas sets the amount of replicas on the resource.
func (r *rabbitmqCluster) setReplicas(replicas int32) error {
	if err := unstructured.SetNestedField(r.Object, int64(replicas), "spec", "replicas"); err != nil {
		return fmt.Errorf("failed to set spec.replicas for %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	return nil
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
//
// The RabbitmqCluster CR carries a single per-pod spec.resources block, so the saving is that
// request multiplied by the change in replicas. A missing or unparseable value counts as zero so
// resource accounting never breaks a scale operation.
func (r *rabbitmqCluster) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	cpu, _, _ := unstructured.NestedString(r.Object, "spec", "resources", "requests", "cpu")
	memory, _, _ := unstructured.NestedString(r.Object, "spec", "resources", "requests", "memory")

	totalSavedCPU := parseQuantityAsFloat(&cpu) * float64(diffReplicas)
	totalSavedMemory := parseQuantityAsFloat(&memory) * float64(diffReplicas)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Copy creates a deep copy of the workload.
// Must use DeepCopy(): Unstructured wraps a map[string]interface{}, so a struct copy would be
// shallow and ScaleDown() on the copy would mutate the original's spec.replicas.
func (r *rabbitmqCluster) Copy() (Workload, error) {
	if r.Object == nil {
		return nil, newNilUnderlyingObjectError(r.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &rabbitmqCluster{
			Unstructured: r.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (r *rabbitmqCluster) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	rCopy, ok := rswCopy.replicaScaledResource.(*rabbitmqCluster)
	if !ok {
		return nil, newExpectTypeGotTypeError((*rabbitmqCluster)(nil), rswCopy.replicaScaledResource)
	}

	if r.Object == nil || rCopy.Object == nil {
		return nil, newNilUnderlyingObjectError(r.GetKind())
	}

	diff, err := jsondiff.Compare(r.Object, rCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare %s: %w", r.GetKind(), err)
	}

	return diff, nil
}

// Reget regets the workload to ensure the latest state.
func (r *rabbitmqCluster) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, rabbitmqClusterGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: r.GetNamespace(), Name: r.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	r.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (r *rabbitmqCluster) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, r.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	return nil
}
