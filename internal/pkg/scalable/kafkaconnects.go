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

var kafkaConnectGVK = schema.GroupVersionKind{
	Group: "kafka.strimzi.io", Version: "v1", Kind: "KafkaConnect",
}

// kafkaConnect wraps an unstructured KafkaConnect CR. The unstructured approach
// is used because no official Strimzi Go client exists.
type kafkaConnect struct {
	*unstructured.Unstructured
}

// getKafkaConnects is the getResourceFunc for KafkaConnect.
func getKafkaConnects(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   kafkaConnectGVK.Group,
		Version: kafkaConnectGVK.Version,
		Kind:    kafkaConnectGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("strimzi CRD not found in cluster, skipping", "kind", kafkaConnectGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get kafkaconnects: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		item.SetGroupVersionKind(kafkaConnectGVK)
		results = append(results, &replicaScaledWorkload{&kafkaConnect{&item}})
	}

	return results, nil
}

// parseKafkaConnectFromBytes parses the admission review and returns the kafkaconnect wrapped in a Workload.
func parseKafkaConnectFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode kafkaconnect: %w", err)
	}

	return &replicaScaledWorkload{&kafkaConnect{&u}}, nil
}

// getReplicas gets the current amount of replicas of the resource.
// JSON numbers from the Kubernetes API are decoded as float64, so both float64 and int64 are handled.
func (k *kafkaConnect) getReplicas() (values.Replicas, error) {
	val, found, err := unstructured.NestedFieldNoCopy(k.Object, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.replicas for %s %s/%s: %w", k.GetKind(), k.GetNamespace(), k.GetName(), err)
	}

	if !found {
		return nil, newNoReplicasError(k.GetKind(), k.GetName())
	}

	switch v := val.(type) {
	case int64:
		return values.AbsoluteReplicas(int32(v)), nil //nolint:gosec // temporary in-place conversion
	case float64:
		return values.AbsoluteReplicas(int32(v)), nil
	default:
		return nil, fmt.Errorf("unexpected type %T for spec.replicas on %s %s/%s", val, k.GetKind(), k.GetNamespace(), k.GetName())
	}
}

// setReplicas sets the amount of replicas on the resource.
func (k *kafkaConnect) setReplicas(replicas int32) error {
	if err := unstructured.SetNestedField(k.Object, int64(replicas), "spec", "replicas"); err != nil {
		return fmt.Errorf("failed to set spec.replicas for %s %s/%s: %w", k.GetKind(), k.GetNamespace(), k.GetName(), err)
	}

	return nil
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
// Strimzi pod templates are not accessible at this abstraction level, consistent with scaledobjects.go.
func (k *kafkaConnect) getSavedResourcesRequests(_ int32) *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// Copy creates a deep copy of the workload.
// Must use DeepCopy() — Unstructured wraps a map[string]interface{} so a struct copy would be
// shallow, causing ScaleDown() on the copy to mutate the original's spec.replicas.
func (k *kafkaConnect) Copy() (Workload, error) {
	if k.Object == nil {
		return nil, newNilUnderlyingObjectError(k.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &kafkaConnect{
			Unstructured: k.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (k *kafkaConnect) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	kCopy, ok := rswCopy.replicaScaledResource.(*kafkaConnect)
	if !ok {
		return nil, newExpectTypeGotTypeError((*kafkaConnect)(nil), rswCopy.replicaScaledResource)
	}

	if k.Object == nil || kCopy.Object == nil {
		return nil, newNilUnderlyingObjectError(k.GetKind())
	}

	diff, err := jsondiff.Compare(k.Object, kCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare %s: %w", k.GetKind(), err)
	}

	return diff, nil
}

// Reget regets the workload to ensure the latest state.
func (k *kafkaConnect) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	fresh.SetGroupVersionKind(kafkaConnectGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: k.GetNamespace(), Name: k.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", k.GetKind(), k.GetNamespace(), k.GetName(), err)
	}

	k.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (k *kafkaConnect) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, k.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", k.GetKind(), k.GetNamespace(), k.GetName(), err)
	}

	return nil
}
