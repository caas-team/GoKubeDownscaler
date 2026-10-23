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

const (
	// mariaDBDefaultReplicas mirrors the CRD default for spec.replicas.
	mariaDBDefaultReplicas = 1
	// mariaDBHAMinReplicas is the lowest spec.replicas the operator's webhook accepts with HA enabled.
	mariaDBHAMinReplicas = 2
)

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var mariaDBGVK = schema.GroupVersionKind{
	Group: mariaDBGroup, Version: mariaDBVersion, Kind: mariaDBKind,
}

// mariaDB wraps an unstructured MariaDB CR from the mariadb-operator. It is replica-shaped:
//   - standalone: on spec.replicas, which the operator propagates to its StatefulSet, including 0.
//   - Galera: the webhook rejects spec.replicas below 2, so scaling down sets spec.suspend instead
//     (freezing the operator) and the StatefulSet is scaled to 0 as a child. On wake the operator
//     restores the StatefulSet and runs its full-cluster Galera recovery.
//
// The metrics exporter Deployment is never scaled by the operator and is always handled as a child.
type mariaDB struct {
	*unstructured.Unstructured
}

// getMariaDBs is the getResourceFunc for MariaDB.
func getMariaDBs(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   mariaDBGVK.Group,
		Version: mariaDBGVK.Version,
		Kind:    mariaDBGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("mariadb-operator CRD not found in cluster, skipping", "kind", mariaDBGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get mariadbs: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, mariaDBGVK)
		results = append(results, &replicaScaledWorkload{&mariaDB{&item}})
	}

	return results, nil
}

// parseMariaDBFromBytes parses the admission review and returns the mariadb wrapped in a Workload.
func parseMariaDBFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode mariadb: %w", err)
	}

	return &replicaScaledWorkload{&mariaDB{&u}}, nil
}

// getReplicas gets the current amount of replicas of the resource.
//
// A scaled-down Galera MariaDB (suspended) reports 0, its spec.replicas being left untouched.
func (m *mariaDB) getReplicas() (values.Replicas, error) {
	if m.isGaleraEnabled() && m.isSuspended() {
		return values.AbsoluteReplicas(0), nil
	}

	val, found, err := unstructured.NestedFieldNoCopy(m.Object, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.replicas for %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	if !found {
		return values.AbsoluteReplicas(mariaDBDefaultReplicas), nil
	}

	replicas, ok := unstructuredReplicasToInt32(val)
	if !ok {
		return nil, newUnexpectedReplicasTypeError(val, m.GetKind(), m.GetNamespace(), m.GetName())
	}

	return values.AbsoluteReplicas(replicas), nil
}

// setReplicas sets the amount of replicas on the resource.
//
// With HA enabled the operator rejects spec.replicas below 2. A Galera MariaDB is scaled down
// through spec.suspend instead, unless its automatic cluster recovery is disabled, since nothing would
// then bootstrap the cluster on wake. Any other target below 2 is refused for HA MariaDBs.
func (m *mariaDB) setReplicas(replicas int32) error {
	if m.isHAEnabled() && replicas < mariaDBHAMinReplicas {
		if replicas != 0 || !m.isGaleraEnabled() {
			return newScalingRefusedError(m.GetKind(), m.GetNamespace(), m.GetName(),
				fmt.Sprintf("galera or replication is enabled, the operator rejects spec.replicas below %d", mariaDBHAMinReplicas))
		}

		if !m.isGaleraRecoveryEnabled() {
			return newScalingRefusedError(m.GetKind(), m.GetNamespace(), m.GetName(),
				"galera recovery is disabled, the cluster could not be bootstrapped on wake")
		}

		return m.setSuspend(true)
	}

	if m.isGaleraEnabled() {
		if err := m.setSuspend(false); err != nil {
			return err
		}
	}

	if err := unstructured.SetNestedField(m.Object, int64(replicas), "spec", "replicas"); err != nil {
		return fmt.Errorf("failed to set spec.replicas for %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	return nil
}

func (m *mariaDB) setSuspend(suspend bool) error {
	if err := unstructured.SetNestedField(m.Object, suspend, "spec", "suspend"); err != nil {
		return fmt.Errorf("failed to set spec.suspend for %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	return nil
}

// GetChildren returns the workloads owned by this MariaDB that the operator does not scale itself:
//   - the metrics exporter Deployment, only reconciled by the operator while the MariaDB is Ready,
//     so it stays scaled down while the database is down instead of crash-looping.
//   - for Galera, the StatefulSet, since scaling down only suspends the operator.
func (m *mariaDB) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	deployments, err := clientsets.Kubernetes.AppsV1().Deployments(m.GetNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments for mariadb %s/%s: %w", m.GetNamespace(), m.GetName(), err)
	}

	children := m.ownedDeployments(deployments.Items)

	if !m.isGaleraEnabled() {
		return children, nil
	}

	statefulSets, err := clientsets.Kubernetes.AppsV1().StatefulSets(m.GetNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets for mariadb %s/%s: %w", m.GetNamespace(), m.GetName(), err)
	}

	return append(children, m.ownedStatefulSets(statefulSets.Items)...), nil
}

func (m *mariaDB) ownedDeployments(deployments []appsv1.Deployment) []Workload {
	results := make([]Workload, 0, 1)

	for i := range deployments {
		dep := &deployments[i]
		if !m.owns(dep) {
			continue
		}

		setGroupVersionKindIfEmpty(dep, appsv1.SchemeGroupVersion.WithKind(deploymentKind))
		results = append(results, &replicaScaledWorkload{&deployment{dep}})
	}

	return results
}

func (m *mariaDB) ownedStatefulSets(statefulSets []appsv1.StatefulSet) []Workload {
	results := make([]Workload, 0, 1)

	for i := range statefulSets {
		sts := &statefulSets[i]
		if !m.owns(sts) {
			continue
		}

		setGroupVersionKindIfEmpty(sts, appsv1.SchemeGroupVersion.WithKind(statefulSetKind))
		results = append(results, &replicaScaledWorkload{&statefulSet{sts}})
	}

	return results
}

func (m *mariaDB) owns(obj metav1.Object) bool {
	for _, owner := range obj.GetOwnerReferences() {
		if owner.Controller == nil || !*owner.Controller {
			continue
		}

		if owner.Kind == mariaDBKind && owner.Name == m.GetName() {
			return true
		}
	}

	return false
}

// MustScaleChildren reports that the children must always be scaled alongside the MariaDB,
// regardless of the scale-children scope setting (see GetChildren).
func (m *mariaDB) MustScaleChildren() bool {
	return true
}

func (m *mariaDB) isHAEnabled() bool {
	replication, _, _ := unstructured.NestedBool(m.Object, "spec", "replication", "enabled")

	return m.isGaleraEnabled() || replication
}

func (m *mariaDB) isGaleraEnabled() bool {
	galera, _, _ := unstructured.NestedBool(m.Object, "spec", "galera", "enabled")

	return galera
}

// isGaleraRecoveryEnabled reports whether the operator will bootstrap the Galera cluster after a full
// stop. Like the operator, an absent recovery block defaults to enabled, but a present one must say so.
func (m *mariaDB) isGaleraRecoveryEnabled() bool {
	recovery, found, _ := unstructured.NestedMap(m.Object, "spec", "galera", "recovery")
	if !found {
		return true
	}

	enabled, _ := recovery["enabled"].(bool)

	return enabled
}

func (m *mariaDB) isSuspended() bool {
	suspended, _, _ := unstructured.NestedBool(m.Object, "spec", "suspend")

	return suspended
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
//
// MariaDB carries the per-instance resources of the mariadb container in spec.resources;
// a missing or unparseable value counts as zero.
func (m *mariaDB) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	cpuReq, _, _ := unstructured.NestedString(m.Object, "spec", "resources", "requests", "cpu")
	memReq, _, _ := unstructured.NestedString(m.Object, "spec", "resources", "requests", "memory")

	totalCPU := parseQuantityAsFloat(&cpuReq)
	totalMemory := parseQuantityAsFloat(&memReq)

	return metrics.NewSavedResources(totalCPU*float64(diffReplicas), totalMemory*float64(diffReplicas))
}

// Copy creates a deep copy of the workload.
func (m *mariaDB) Copy() (Workload, error) {
	if m.Object == nil {
		return nil, newNilUnderlyingObjectError(m.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &mariaDB{
			Unstructured: m.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (m *mariaDB) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	mCopy, ok := rswCopy.replicaScaledResource.(*mariaDB)
	if !ok {
		return nil, newExpectTypeGotTypeError((*mariaDB)(nil), rswCopy.replicaScaledResource)
	}

	if m.Object == nil || mCopy.Object == nil {
		return nil, newNilUnderlyingObjectError(m.GetKind())
	}

	diff, err := jsondiff.Compare(m.Object, mCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare %s: %w", m.GetKind(), err)
	}

	return diff, nil
}

// Reget regets the workload to ensure the latest state.
func (m *mariaDB) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, mariaDBGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: m.GetNamespace(), Name: m.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	m.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (m *mariaDB) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, m.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	return nil
}
