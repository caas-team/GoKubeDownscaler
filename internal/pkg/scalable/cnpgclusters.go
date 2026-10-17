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

// cnpgHibernationAnnotation declaratively hibernates a CloudNativePG Cluster. CNPG has no
// StatefulSet - it manages instance Pods directly - and deletes every instance Pod itself
// (preserving PVCs) once this annotation is "on", recreating them and re-electing a primary once
// it's "off", with no bookkeeping needed from the caller.
const (
	cnpgHibernationAnnotation = "cnpg.io/hibernation"
	cnpgHibernationValueOn    = "on"
	cnpgHibernationValueOff   = "off"
)

//nolint:gochecknoglobals // package-level GVK required for unstructured client
var cnpgClusterGVK = schema.GroupVersionKind{
	Group: cnpgGroup, Version: cnpgVersion, Kind: cnpgClusterKind,
}

// cnpgCluster wraps an unstructured CloudNativePG Cluster CR. The unstructured approach avoids
// importing the operator's API module. It is suspend-shaped on the cnpg.io/hibernation
// annotation; no child workload needs to be scaled.
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
			slog.Warn("cloudnativepg CRD not found in cluster, skipping", "kind", cnpgClusterGVK.Kind, "error", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get cnpgclusters: %w", err)
	}

	results := make([]Workload, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		setGroupVersionKindIfEmpty(&item, cnpgClusterGVK)
		results = append(results, &valueScaledWorkload{&cnpgCluster{&item}})
	}

	return results, nil
}

// parseCnpgClusterFromBytes parses the admission review and returns the cnpgcluster wrapped in a Workload.
func parseCnpgClusterFromBytes(rawObject []byte) (Workload, error) {
	var u unstructured.Unstructured
	if err := json.Unmarshal(rawObject, &u); err != nil {
		return nil, fmt.Errorf("failed to decode cnpgcluster: %w", err)
	}

	return &valueScaledWorkload{&cnpgCluster{&u}}, nil
}

// getValue gets the current value of the hibernation annotation and the target downscale state for it.
// Nothing can fail reading an annotation, so err is always nil.
//
//nolint:nonamedreturns // clarifies which value is current and which is the downscale target
func (c *cnpgCluster) getValue() (currentValue, targetDownscaleState values.Replicas, err error) {
	current := c.GetAnnotations()[cnpgHibernationAnnotation] == cnpgHibernationValueOn

	return values.BooleanReplicas(current), values.BooleanReplicas(true), nil
}

// setValue sets the hibernation annotation to request or clear hibernation. The annotation is
// always written explicitly (never removed) since CNPG treats an absent annotation the same as
// "off".
func (c *cnpgCluster) setValue(value values.Replicas) error {
	suspend, err := value.AsBool()
	if err != nil {
		return fmt.Errorf("failed to convert target value to bool: %w", err)
	}

	annotations := c.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[cnpgHibernationAnnotation] = cnpgHibernationValueOff
	if suspend {
		annotations[cnpgHibernationAnnotation] = cnpgHibernationValueOn
	}

	c.SetAnnotations(annotations)

	return nil
}

// logUpscaleSuccessful logs the hibernation annotation transition for an upscale.
func (c *cnpgCluster) logUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("scaled up", "hibernation", c, summary, dryRun, logger)
}

// logDownscaleSuccessful logs the hibernation annotation transition for a downscale.
func (c *cnpgCluster) logDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("scaled down", "hibernation", c, summary, dryRun, logger)
}

// getSavedResourcesRequests returns the saved CPU and memory requests.
//
// Hibernation stops every instance, so this is the per-instance request from spec.resources
// multiplied by spec.instances; a missing or unparseable value counts as zero.
func (c *cnpgCluster) getSavedResourcesRequests() *metrics.SavedResources {
	instances, _, _ := unstructured.NestedInt64(c.Object, "spec", "instances")

	cpuReq, _, _ := unstructured.NestedString(c.Object, "spec", "resources", "requests", "cpu")
	memReq, _, _ := unstructured.NestedString(c.Object, "spec", "resources", "requests", "memory")

	totalCPU := parseQuantityAsFloat(&cpuReq) * float64(instances)
	totalMemory := parseQuantityAsFloat(&memReq) * float64(instances)

	return metrics.NewSavedResources(totalCPU, totalMemory)
}

// Copy creates a deep copy of the workload.
func (c *cnpgCluster) Copy() (Workload, error) {
	if c.Object == nil {
		return nil, newNilUnderlyingObjectError(c.GetKind())
	}

	return &valueScaledWorkload{
		valueScaledResource: &cnpgCluster{
			Unstructured: c.DeepCopy(),
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen // short names are ok for the workflow of this function
func (c *cnpgCluster) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	vswCopy, ok := workloadCopy.(*valueScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*valueScaledWorkload)(nil), workloadCopy)
	}

	cCopy, ok := vswCopy.valueScaledResource.(*cnpgCluster)
	if !ok {
		return nil, newExpectTypeGotTypeError((*cnpgCluster)(nil), vswCopy.valueScaledResource)
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
