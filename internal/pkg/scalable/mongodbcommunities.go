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

// mongoLastSuccessfulConfigurationAnnotation is written by the MongoDB Community
// operator once it has successfully applied a configuration. Its presence means
// the operator has a known-good config it can reconcile back to.
//
// This scaler REFUSES to park a MongoDBCommunity that lacks this annotation
// (fail closed): parking such a CR to spec.members: 0 before the operator has
// ever reconciled a good config risks the operator being unable to bring it back
// (D9/D10). This safety rule is why the Mongo scaler is fork-only and not an
// upstream contribution.
const mongoLastSuccessfulConfigurationAnnotation = "mongodb.com/v1.lastSuccessfulConfiguration"

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var mongoDBCommunityGVK = schema.GroupVersionKind{
	Group: mongoDBCommunityGroup, Version: mongoDBCommunityVersion, Kind: mongoDBCommunityKind,
}

// mongoDBCommunity wraps an unstructured MongoDBCommunity CR from the MongoDB
// Community operator. The unstructured approach avoids importing the operator's
// API module. It is replica-shaped on spec.members; the operator reconciles the
// child StatefulSet's replica count from that field, so the StatefulSet is left
// to the operator (the kind is in supportedOwnerKinds).
type mongoDBCommunity struct {
	*unstructured.Unstructured
}

// getMongoDBCommunities is the getResourceFunc for MongoDBCommunity.
func getMongoDBCommunities(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   mongoDBCommunityGVK.Group,
		Version: mongoDBCommunityGVK.Version,
		Kind:    mongoDBCommunityGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("mongodb community operator CRD not found in cluster, skipping", "kind", mongoDBCommunityGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get mongodbcommunities: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, mongoDBCommunityGVK)
		results = append(results, &replicaScaledWorkload{&mongoDBCommunity{&item}})
	}

	return results, nil
}

// parseMongoDBCommunityFromBytes parses the admission review and returns the mongodbcommunity wrapped in a Workload.
func parseMongoDBCommunityFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode mongodbcommunity: %w", err)
	}

	return &replicaScaledWorkload{&mongoDBCommunity{&u}}, nil
}

// getReplicas gets the current amount of replica-set members of the resource.
func (m *mongoDBCommunity) getReplicas() (values.Replicas, error) {
	val, found, err := unstructured.NestedFieldNoCopy(m.Object, "spec", "members")
	if err != nil {
		return nil, fmt.Errorf("failed to get spec.members for %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	if !found {
		return nil, newNoReplicasError(m.GetKind(), m.GetName())
	}

	replicas, ok := unstructuredReplicasToInt32(val)
	if !ok {
		return nil, newUnexpectedReplicasTypeError(val, m.GetKind(), m.GetNamespace(), m.GetName())
	}

	return values.AbsoluteReplicas(replicas), nil
}

// setReplicas sets the amount of replica-set members on the resource.
//
// It refuses to scale DOWN (park) a CR that has never recorded a successful
// configuration (the lastSuccessfulConfiguration annotation is absent), so the
// downscaler never parks a Mongo the operator could not bring back. Scaling up
// is always allowed.
func (m *mongoDBCommunity) setReplicas(replicas int32) error {
	if replicas == 0 {
		if _, ok := m.GetAnnotations()[mongoLastSuccessfulConfigurationAnnotation]; !ok {
			return newParkRefusedError(m.GetKind(), m.GetNamespace(), m.GetName(),
				fmt.Sprintf("annotation %q is absent, operator has no known-good config to restore", mongoLastSuccessfulConfigurationAnnotation))
		}
	}

	if err := unstructured.SetNestedField(m.Object, int64(replicas), "spec", "members"); err != nil {
		return fmt.Errorf("failed to set spec.members for %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	return nil
}

// GetChildren returns the StatefulSet the MongoDB Community operator created for
// this resource, wrapped as a replica-scaled workload so the downscaler scales
// it to 0 on park. The operator short-circuits reconciliation once spec.members
// is 0 (validateSpec rejects the spec before the StatefulSet is written), so it
// will NOT scale the StatefulSet down itself — the scaler must. The StatefulSet
// is matched by an ownerReference back to this MongoDBCommunity.
func (m *mongoDBCommunity) GetChildren(ctx context.Context, clientsets *Clientsets) ([]Workload, error) {
	list, err := clientsets.Kubernetes.AppsV1().StatefulSets(m.GetNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets for mongodbcommunity %s/%s: %w", m.GetNamespace(), m.GetName(), err)
	}

	results := make([]Workload, 0, 1)

	for i := range list.Items {
		sts := &list.Items[i]

		owned := false

		for _, owner := range sts.GetOwnerReferences() {
			if owner.Kind == mongoDBCommunityKind && owner.Name == m.GetName() {
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
// MongoDBCommunity carries per-member resources under the mongod container in
// spec.statefulSet.spec.template; a missing or unparseable value counts as zero.
func (m *mongoDBCommunity) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	var totalCPU, totalMemory float64

	containers, _, _ := unstructured.NestedSlice(m.Object, "spec", "statefulSet", "spec", "template", "spec", "containers")
	for _, c := range containers {
		container, ok := c.(map[string]any)
		if !ok {
			continue
		}

		cpuReq, _, _ := unstructured.NestedString(container, "resources", "requests", "cpu")
		memReq, _, _ := unstructured.NestedString(container, "resources", "requests", "memory")
		totalCPU += parseQuantityAsFloat(&cpuReq)
		totalMemory += parseQuantityAsFloat(&memReq)
	}

	return metrics.NewSavedResources(totalCPU*float64(diffReplicas), totalMemory*float64(diffReplicas))
}

// Copy creates a deep copy of the workload.
func (m *mongoDBCommunity) Copy() (Workload, error) {
	if m.Object == nil {
		return nil, newNilUnderlyingObjectError(m.GetKind())
	}

	return &replicaScaledWorkload{
		replicaScaledResource: &mongoDBCommunity{
			Unstructured: m.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (m *mongoDBCommunity) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	mCopy, ok := rswCopy.replicaScaledResource.(*mongoDBCommunity)
	if !ok {
		return nil, newExpectTypeGotTypeError((*mongoDBCommunity)(nil), rswCopy.replicaScaledResource)
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
func (m *mongoDBCommunity) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, mongoDBCommunityGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: m.GetNamespace(), Name: m.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	m.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (m *mongoDBCommunity) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, m.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", m.GetKind(), m.GetNamespace(), m.GetName(), err)
	}

	return nil
}
