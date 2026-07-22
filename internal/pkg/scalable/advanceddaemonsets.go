//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	kruisev1beta1 "github.com/openkruise/kruise/apis/apps/v1beta1"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getAdvancedDaemonSets is the getResourceFunc for advanced daemonsets.
func getAdvancedDaemonSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	daemonsets, err := clientsets.Kruise.AppsV1beta1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get advanceddaemonsets: %w", err)
	}

	results := make([]Workload, 0, len(daemonsets.Items))
	for i := range daemonsets.Items {
		setGroupVersionKindIfEmpty(&daemonsets.Items[i], kruisev1beta1.SchemeGroupVersion.WithKind("AdvancedDaemonSet"))
		results = append(results, &nodeSelectorScaledWorkload{nodeSelectorScaledResource: &advancedDaemonSet{&daemonsets.Items[i]}})
	}

	return results, nil
}

// parseAdvancedDaemonSetFromBytes parses the admission review and returns the advanced daemonset.
func parseAdvancedDaemonSetFromBytes(rawObject []byte) (Workload, error) {
	var ds kruisev1beta1.DaemonSet
	if err := json.Unmarshal(rawObject, &ds); err != nil {
		return nil, fmt.Errorf("failed to decode advanceddaemonset: %w", err)
	}

	return &nodeSelectorScaledWorkload{nodeSelectorScaledResource: &advancedDaemonSet{&ds}}, nil
}

// advancedDaemonSet is a wrapper for daemonset.v1beta1.apps.kruise.io to implement the nodeSelectorScaledResource interface.
type advancedDaemonSet struct {
	*kruisev1beta1.DaemonSet
}

// getNodeSelector gets the node selector from the underlying advanced DaemonSet.
func (d *advancedDaemonSet) getNodeSelector() map[string]string {
	return d.Spec.Template.Spec.NodeSelector
}

// setNodeSelector sets the node selector on the underlying advanced DaemonSet.
func (d *advancedDaemonSet) setNodeSelector(nodeSelector map[string]string) {
	d.Spec.Template.Spec.NodeSelector = nodeSelector
}

// Reget regets the resource from the Kubernetes API.
func (d *advancedDaemonSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	d.DaemonSet, err = clientsets.Kruise.AppsV1beta1().DaemonSets(d.Namespace).Get(ctx, d.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get advanceddaemonset: %w", err)
	}

	return nil
}

// getResourcesRequests calculates the total saved resources requests when downscaling the DaemonSet.
func (d *advancedDaemonSet) getResourcesRequests(_ int32) *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range d.Spec.Template.Spec.Containers {
		container := &d.Spec.Template.Spec.Containers[i]
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	totalSavedCPU *= float64(d.Status.CurrentNumberScheduled)
	totalSavedMemory *= float64(d.Status.CurrentNumberScheduled)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (d *advancedDaemonSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1beta1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update advanceddaemonset: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be an advancedDaemonSet.
func (d *advancedDaemonSet) Copy() (Workload, error) {
	if d.DaemonSet == nil {
		return nil, newNilUnderlyingObjectError(d.Kind)
	}

	copied := d.DeepCopy()

	return &nodeSelectorScaledWorkload{
		nodeSelectorScaledResource: &advancedDaemonSet{
			DaemonSet: copied,
		},
	}, nil
}

// Compare compares two advancedDaemonSet resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (d *advancedDaemonSet) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	nswCopy, ok := workloadCopy.(*nodeSelectorScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*nodeSelectorScaledWorkload)(nil), workloadCopy)
	}

	dsCopy, ok := nswCopy.nodeSelectorScaledResource.(*advancedDaemonSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*advancedDaemonSet)(nil), nswCopy.nodeSelectorScaledResource)
	}

	if d.DaemonSet == nil || dsCopy.DaemonSet == nil {
		return nil, newNilUnderlyingObjectError(d.Kind)
	}

	diff, err := jsondiff.Compare(d.DaemonSet, dsCopy.DaemonSet)
	if err != nil {
		return nil, fmt.Errorf("failed to compare advanceddaemonsets: %w", err)
	}

	return diff, nil
}
