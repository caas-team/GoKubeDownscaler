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

var (
	kafkaConnectGVK = schema.GroupVersionKind{
		Group: "kafka.strimzi.io", Version: "v1beta2", Kind: "KafkaConnect",
	}
	kafkaMirrorMaker2GVK = schema.GroupVersionKind{
		Group: "kafka.strimzi.io", Version: "v1beta2", Kind: "KafkaMirrorMaker2",
	}
	kafkaBridgeGVK = schema.GroupVersionKind{
		Group: "kafka.strimzi.io", Version: "v1beta2", Kind: "KafkaBridge",
	}
)

// strimziWorkload wraps an unstructured Strimzi CR that exposes spec.replicas.
// It is used for KafkaConnect, KafkaMirrorMaker2, and KafkaBridge — all of which
// scale by setting spec.replicas on the CR directly.
//
// The unstructured approach is used because no official Strimzi Go client exists.
// See: docs/brainstorms/2026-03-30-strimzi-workload-support-requirements.md
type strimziWorkload struct {
	*unstructured.Unstructured
	gvk schema.GroupVersionKind
}

// getStrimziWorkloads lists all resources of the given GVK in the namespace.
// It returns an empty list (with a Warn log) if the Strimzi CRD is not installed,
// so the scan loop continues for other resource types.
func getStrimziWorkloads(namespace string, gvk schema.GroupVersionKind, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List", // controller-runtime requires the "List" suffix
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("strimzi CRD not found in cluster, skipping", "kind", gvk.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get %ss: %w", gvk.Kind, err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		item.SetGroupVersionKind(gvk) // defensive: stamp GVK on each item
		results = append(results, &replicaScaledWorkload{&strimziWorkload{&item, gvk}})
	}

	return results, nil
}

// getKafkaConnects is the getResourceFunc for KafkaConnect.
func getKafkaConnects(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	return getStrimziWorkloads(namespace, kafkaConnectGVK, clientsets, ctx)
}

// getKafkaMirrorMaker2s is the getResourceFunc for KafkaMirrorMaker2.
func getKafkaMirrorMaker2s(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	return getStrimziWorkloads(namespace, kafkaMirrorMaker2GVK, clientsets, ctx)
}

// getKafkaBridges is the getResourceFunc for KafkaBridge.
func getKafkaBridges(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	return getStrimziWorkloads(namespace, kafkaBridgeGVK, clientsets, ctx)
}

// parseKafkaConnectFromBytes parses the admission review and returns the kafkaconnect wrapped in a Workload.
// The GVK is already populated from the API server's raw bytes — no manual stamping needed.
func parseKafkaConnectFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode kafkaconnect: %w", err)
	}

	return &replicaScaledWorkload{&strimziWorkload{&u, kafkaConnectGVK}}, nil
}

// parseKafkaMirrorMaker2FromBytes parses the admission review and returns the kafkamirrormaker2 wrapped in a Workload.
func parseKafkaMirrorMaker2FromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode kafkamirrormaker2: %w", err)
	}

	return &replicaScaledWorkload{&strimziWorkload{&u, kafkaMirrorMaker2GVK}}, nil
}

// parseKafkaBridgeFromBytes parses the admission review and returns the kafkabridge wrapped in a Workload.
func parseKafkaBridgeFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode kafkabridge: %w", err)
	}

	return &replicaScaledWorkload{&strimziWorkload{&u, kafkaBridgeGVK}}, nil
}

// getReplicas gets the current amount of replicas of the resource.
// JSON numbers from the Kubernetes API are decoded as float64, so both float64 and int64 are handled.
func (s *strimziWorkload) getReplicas() (values.Replicas, error) {
	val, found, err := unstructured.NestedFieldNoCopy(s.Object, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.replicas for %s %s/%s: %w", s.GetKind(), s.GetNamespace(), s.GetName(), err)
	}

	if !found {
		return nil, newNoReplicasError(s.GetKind(), s.GetName())
	}

	switch v := val.(type) {
	case int64:
		return values.AbsoluteReplicas(int32(v)), nil //nolint:gosec // temporary in-place conversion
	case float64:
		return values.AbsoluteReplicas(int32(v)), nil //nolint:gosec // temporary in-place conversion
	default:
		return nil, fmt.Errorf("unexpected type %T for spec.replicas on %s %s/%s", val, s.GetKind(), s.GetNamespace(), s.GetName())
	}
}

// setReplicas sets the amount of replicas on the resource.
func (s *strimziWorkload) setReplicas(replicas int32) error {
	if err := unstructured.SetNestedField(s.Object, int64(replicas), "spec", "replicas"); err != nil {
		return fmt.Errorf("failed to set spec.replicas for %s %s/%s: %w", s.GetKind(), s.GetNamespace(), s.GetName(), err)
	}

	return nil
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
// Strimzi pod templates are not accessible at this abstraction level, consistent with scaledobjects.go.
func (s *strimziWorkload) getSavedResourcesRequests(_ int32) *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// Copy creates a deep copy of the workload.
// Must use DeepCopy() — Unstructured wraps a map[string]interface{} so a struct copy would be
// shallow, causing ScaleDown() on the copy to mutate the original's spec.replicas.
func (s *strimziWorkload) Copy() (Workload, error) {
	if s.Object == nil {
		return nil, newNilUnderlyingObjectError(s.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &strimziWorkload{
			Unstructured: s.Unstructured.DeepCopy(),
			gvk:          s.gvk,
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
// The inner Object maps are compared directly, consistent with how typed workloads compare their
// underlying structs.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (s *strimziWorkload) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	sCopy, ok := rswCopy.replicaScaledResource.(*strimziWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*strimziWorkload)(nil), rswCopy.replicaScaledResource)
	}

	if s.Object == nil || sCopy.Object == nil {
		return nil, newNilUnderlyingObjectError(s.GetKind())
	}

	diff, err := jsondiff.Compare(s.Object, sCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare %s: %w", s.GetKind(), err)
	}

	return diff, nil
}

// Reget regets the workload to ensure the latest state.
func (s *strimziWorkload) Reget(clientsets *Clientsets, ctx context.Context) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(s.gvk)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: s.GetNamespace(), Name: s.GetName()}, u)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", s.GetKind(), s.GetNamespace(), s.GetName(), err)
	}

	s.Unstructured = u

	return nil
}

// Update updates the resource with all changes made to it.
func (s *strimziWorkload) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, s.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", s.GetKind(), s.GetNamespace(), s.GetName(), err)
	}

	return nil
}
