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

// cnpgHibernationAnnotation is the CloudNativePG operator's declarative hibernation switch.
// Setting it to cnpgHibernationOn deletes the instance pods (replicas first, primary last) while
// retaining the PVCs; cnpgHibernationOff (or removing the annotation) rehydrates the cluster.
const (
	cnpgHibernationAnnotation = "cnpg.io/hibernation"
	cnpgHibernationOn         = "on"
	cnpgHibernationOff        = "off"
)

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var cnpgClusterGVK = schema.GroupVersionKind{
	Group: cnpgGroup, Version: cnpgVersion, Kind: cnpgClusterKind,
}

// cnpgCluster wraps an unstructured CloudNativePG Cluster CR.
// The unstructured approach is used to avoid importing the operator's API module.
//
// Unlike replica-shaped scalers, CNPG has no spec.replicas the downscaler may drive: the operator
// owns instance count. It exposes hibernation only through the cnpg.io/hibernation annotation, so
// this is a suspend-shaped workload whose "suspend field" is that annotation.
type cnpgCluster struct {
	*unstructured.Unstructured
}

// getCnpgClusters is the getResourceFunc for CNPG Clusters.
func getCnpgClusters(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   cnpgClusterGVK.Group,
		Version: cnpgClusterGVK.Version,
		Kind:    cnpgClusterGVK.Kind + "List",
	})

	if err := clientsets.Client.List(ctx, list, ctrlclient.InNamespace(namespace)); err != nil {
		if apimeta.IsNoMatchError(err) {
			slog.Warn("cnpg operator CRD not found in cluster, skipping", "kind", cnpgClusterGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get cnpgclusters: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, cnpgClusterGVK)
		results = append(results, &suspendScaledWorkload{&cnpgCluster{&item}})
	}

	return results, nil
}

// parseCnpgClusterFromBytes parses the admission review and returns the cnpgcluster wrapped in a Workload.
func parseCnpgClusterFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode cnpgcluster: %w", err)
	}

	return &suspendScaledWorkload{&cnpgCluster{&u}}, nil
}

// getSuspend gets the current hibernation state of the cluster and the target downscale state for it.
//
//nolint:nonamedreturns // named returns document the (current, target) contract of the interface
func (c *cnpgCluster) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	current := c.GetAnnotations()[cnpgHibernationAnnotation] == cnpgHibernationOn

	currentValue = values.BooleanReplicas(current)
	targetDownscaleState = values.BooleanReplicas(true)

	return currentValue, targetDownscaleState
}

// setSuspend sets the hibernation annotation on the cluster.
// true hibernates (cnpg.io/hibernation: on), false rehydrates (cnpg.io/hibernation: off).
func (c *cnpgCluster) setSuspend(suspend bool) {
	value := cnpgHibernationOff
	if suspend {
		value = cnpgHibernationOn
	}

	annotations := c.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[cnpgHibernationAnnotation] = value
	c.SetAnnotations(annotations)
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
//
// Hibernation removes every instance pod, so the saving is the per-instance request multiplied by
// the instance count. A missing or unparseable value counts as zero so resource accounting never
// breaks a scale operation.
func (c *cnpgCluster) getSavedResourcesRequests() *metrics.SavedResources {
	instances, _, _ := unstructured.NestedInt64(c.Object, "spec", "instances")
	if instances <= 0 {
		instances = 1
	}

	cpu, _, _ := unstructured.NestedString(c.Object, "spec", "resources", "requests", "cpu")
	memory, _, _ := unstructured.NestedString(c.Object, "spec", "resources", "requests", "memory")

	totalSavedCPU := parseQuantityAsFloat(&cpu) * float64(instances)
	totalSavedMemory := parseQuantityAsFloat(&memory) * float64(instances)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Copy creates a deep copy of the workload.
// Must use DeepCopy(): Unstructured wraps a map[string]interface{}, so a struct copy would be
// shallow and a mutation on the copy would leak back into the original.
func (c *cnpgCluster) Copy() (Workload, error) {
	if c.Object == nil {
		return nil, newNilUnderlyingObjectError(c.GetKind())
	}

	return &suspendScaledWorkload{
		suspendScaledResource: &cnpgCluster{
			Unstructured: c.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (c *cnpgCluster) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	cCopy, ok := sswCopy.suspendScaledResource.(*cnpgCluster)
	if !ok {
		return nil, newExpectTypeGotTypeError((*cnpgCluster)(nil), sswCopy.suspendScaledResource)
	}

	if c.Object == nil || cCopy.Object == nil {
		return nil, newNilUnderlyingObjectError(c.GetKind())
	}

	diff, err := jsondiff.Compare(c.Object, cCopy.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to compare %s: %w", c.GetKind(), err)
	}

	return diff, nil
}

// Reget regets the workload to ensure the latest state.
func (c *cnpgCluster) Reget(clientsets *Clientsets, ctx context.Context) error {
	fresh := &unstructured.Unstructured{}
	setGroupVersionKindIfEmpty(fresh, cnpgClusterGVK)

	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: c.GetNamespace(), Name: c.GetName()}, fresh)
	if err != nil {
		return fmt.Errorf("failed to get %s %s/%s: %w", c.GetKind(), c.GetNamespace(), c.GetName(), err)
	}

	c.Unstructured = fresh

	return nil
}

// Update updates the resource with all changes made to it.
func (c *cnpgCluster) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, c.Unstructured)
	if err != nil {
		return fmt.Errorf("failed to update %s %s/%s: %w", c.GetKind(), c.GetNamespace(), c.GetName(), err)
	}

	return nil
}
