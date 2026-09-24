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
	appsv1 "k8s.io/api/apps/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// The OT-Container-Kit redis-operator exposes two child-less-to-us CRs that VCP
// runs: RedisReplication and RedisSentinel. Both share the same group/version and
// are scaled by spec.clusterSize (NOT spec.replicas). The operator reconciles the
// child StatefulSet's replica count from clusterSize and owns it by
// ownerReference, so both kinds go in supportedOwnerKinds and the StatefulSets are
// left to the operator.
//
//nolint:gochecknoglobals // package-level GVKs required for the unstructured client
var (
	redisReplicationGVK = schema.GroupVersionKind{Group: redisGroup, Version: redisVersion, Kind: redisReplicationKind}
	redisSentinelGVK    = schema.GroupVersionKind{Group: redisGroup, Version: redisVersion, Kind: redisSentinelKind}
)

// redisWorkload wraps an unstructured redis-operator CR (RedisReplication or
// RedisSentinel). It is parameterized by its GVK so one implementation serves
// both kinds. It is replica-shaped on spec.clusterSize.
type redisWorkload struct {
	*unstructured.Unstructured
	gvk schema.GroupVersionKind
}

// getRedisReplications is the getResourceFunc for RedisReplication.
func getRedisReplications(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	return getRedisWorkloads(namespace, clientsets, ctx, redisReplicationGVK)
}

// getRedisSentinels is the getResourceFunc for RedisSentinel.
func getRedisSentinels(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	return getRedisWorkloads(namespace, clientsets, ctx, redisSentinelGVK)
}

func getRedisWorkloads(namespace string, clientsets *Clientsets, ctx context.Context, gvk schema.GroupVersionKind) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("redis operator CRD not found in cluster, skipping", "kind", gvk.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get %s: %w", gvk.Kind, err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, gvk)
		results = append(results, &replicaScaledWorkload{&redisWorkload{Unstructured: &item, gvk: gvk}})
	}

	return results, nil
}

// parseRedisReplicationFromBytes parses the admission review for a RedisReplication.
func parseRedisReplicationFromBytes(rawObject []byte) (Workload, error) {
	return parseRedisFromBytes(rawObject, redisReplicationGVK)
}

// parseRedisSentinelFromBytes parses the admission review for a RedisSentinel.
func parseRedisSentinelFromBytes(rawObject []byte) (Workload, error) {
	return parseRedisFromBytes(rawObject, redisSentinelGVK)
}

func parseRedisFromBytes(rawObject []byte, gvk schema.GroupVersionKind) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", gvk.Kind, err)
	}

	return &replicaScaledWorkload{&redisWorkload{Unstructured: &u, gvk: gvk}}, nil
}

// getReplicas gets the current clusterSize of the resource.
func (r *redisWorkload) getReplicas() (values.Replicas, error) {
	val, found, err := unstructured.NestedFieldNoCopy(r.Object, "spec", "clusterSize")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.clusterSize for %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
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

// setReplicas sets the clusterSize on the resource.
//
// RedisSentinel's CRD enforces clusterSize >= 1, so it can never be driven to 0
// through the CR. For sentinel the clusterSize is clamped to the minimum, and the
// real park is carried by scaling its child StatefulSet to 0 (see GetChildren).
// RedisReplication has no such floor and parks to 0 on the CR directly.
func (r *redisWorkload) setReplicas(replicas int32) error {
	target := int64(replicas)
	if r.gvk.Kind == redisSentinelKind && target < 1 {
		target = 1
	}

	if err := unstructured.SetNestedField(r.Object, target, "spec", "clusterSize"); err != nil {
		return fmt.Errorf("failed to set spec.clusterSize for %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	return nil
}

// GetChildren returns the StatefulSet owned by a RedisSentinel so the downscaler
// scales it to 0 on park. This is only needed for RedisSentinel: its CRD forbids
// clusterSize 0, so the CR cannot express a full park and the child StatefulSet
// carries it instead. RedisReplication returns no children — its operator scales
// its own StatefulSet from clusterSize 0.
func (r *redisWorkload) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	if r.gvk.Kind != redisSentinelKind {
		return nil, nil
	}

	list, err := clientsets.Kubernetes.AppsV1().StatefulSets(r.GetNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets for %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	results := make([]Workload, 0, 1)

	for i := range list.Items {
		sts := &list.Items[i]

		owned := false

		for _, owner := range sts.GetOwnerReferences() {
			if owner.Kind == redisSentinelKind && owner.Name == r.GetName() {
				owned = true
				break
			}
		}

		if !owned {
			continue
		}

		setGroupVersionKindIfEmpty(sts, appsv1.SchemeGroupVersion.WithKind(statefulSetKind))
		results = append(results, &replicaScaledWorkload{&statefulSet{sts}})
	}

	return results, nil
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
//
// redis-operator CRs carry per-pod resources under spec.kubernetesConfig.resources;
// the saving is that request multiplied by the change in clusterSize. Missing or
// unparseable values count as zero so accounting never breaks a scale.
func (r *redisWorkload) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	cpu, _, _ := unstructured.NestedString(r.Object, "spec", "kubernetesConfig", "resources", "requests", "cpu")
	memory, _, _ := unstructured.NestedString(r.Object, "spec", "kubernetesConfig", "resources", "requests", "memory")

	totalCPU := parseQuantityAsFloat(&cpu) * float64(diffReplicas)
	totalMemory := parseQuantityAsFloat(&memory) * float64(diffReplicas)

	return metrics.NewSavedResources(totalCPU, totalMemory)
}

// Copy creates a deep copy of the workload.
func (r *redisWorkload) Copy() (Workload, error) {
	if r.Object == nil {
		return nil, newNilUnderlyingObjectError(r.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &redisWorkload{
			Unstructured: r.DeepCopy(),
			gvk:          r.gvk,
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (r *redisWorkload) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	rCopy, ok := rswCopy.replicaScaledResource.(*redisWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*redisWorkload)(nil), rswCopy.replicaScaledResource)
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
func (r *redisWorkload) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, r.gvk)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: r.GetNamespace(), Name: r.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	r.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (r *redisWorkload) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, r.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", r.GetKind(), r.GetNamespace(), r.GetName(), err)
	}

	return nil
}
