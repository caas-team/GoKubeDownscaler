package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	labelMatchNone = "downscaler/match-none"
)

// getDaemonSets is the getResourceFunc for DaemonSets.
func getDaemonSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	daemonsets, err := clientsets.Kubernetes.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonsets: %w", err)
	}

	results := make([]Workload, 0, len(daemonsets.Items))
	for i := range daemonsets.Items {
		results = append(results, &daemonSet{&daemonsets.Items[i]})
	}

	return results, nil
}

// parseDaemonSetFromBytes parses the admission review and returns the daemonset.
func parseDaemonSetFromBytes(rawObject []byte) (Workload, error) {
	var ds appsv1.DaemonSet
	if err := json.Unmarshal(rawObject, &ds); err != nil {
		return nil, fmt.Errorf("failed to decode daemonset: %w", err)
	}

	return &daemonSet{&ds}, nil
}

// daemonSet is a wrapper for daemonset.v1.apps to implement the Workload interface.
type daemonSet struct {
	*appsv1.DaemonSet
}

// ScaleUp scales the resource up.
func (d *daemonSet) ScaleUp() (bool, error) {
	_, err := getOriginalReplicas(d)
	if err != nil {
		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if errors.As(err, &originalReplicasUnsetErr) {
			slog.Debug("original replicas is not set, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())
			return false, nil
		}

		return false, fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	delete(d.Spec.Template.Spec.NodeSelector, labelMatchNone)

	removeOriginalReplicas(d)

	return true, nil
}

// ScaleDown scales the resource down.
func (d *daemonSet) ScaleDown(_ values.Replicas) (*metrics.SavedResources, bool, error) {
	if _, hasLabel := d.Spec.Template.Spec.NodeSelector[labelMatchNone]; hasLabel {
		_, err := getOriginalReplicas(d)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if !errors.As(err, &originalReplicasUnsetErr) {
				return metrics.NewSavedResources(0, 0), false, fmt.Errorf("failed to get original replicas for workload: %w", err)
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())

			return metrics.NewSavedResources(0, 0), false, nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", d.GetName(), "namespace", d.GetNamespace())

		return d.getResourcesRequests(d.Status.DesiredNumberScheduled), false, nil
	}

	if d.Spec.Template.Spec.NodeSelector == nil {
		d.Spec.Template.Spec.NodeSelector = map[string]string{}
	}

	d.Spec.Template.Spec.NodeSelector[labelMatchNone] = "true"

	savedResources := d.getResourcesRequests(d.Status.DesiredNumberScheduled)

	setOriginalReplicas(values.BooleanReplicas(false), d)

	return savedResources, true, nil
}

// Reget regets the resource from the Kubernetes API.
func (d *daemonSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	d.DaemonSet, err = clientsets.Kubernetes.AppsV1().DaemonSets(d.Namespace).Get(ctx, d.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cronjob: %w", err)
	}

	return nil
}

// getResourcesRequests calculates the total saved resources requests when downscaling the DaemonSet.
//

func (d *daemonSet) getResourcesRequests(_ int32) *metrics.SavedResources {
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
func (d *daemonSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a daemonSet.
func (d *daemonSet) Copy() (Workload, error) {
	if d.DaemonSet == nil {
		return nil, newNilUnderlyingObjectError(d.Kind)
	}

	copied := d.DeepCopy()

	return &daemonSet{DaemonSet: copied}, nil
}

// Compare compares two daemonSet resources and returns the differences as a jsondiff.Patch.
func (d *daemonSet) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	dsCopy, ok := workloadCopy.(*daemonSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*daemonSet)(nil), workloadCopy)
	}

	if d.DaemonSet == nil || dsCopy.DaemonSet == nil {
		return nil, newNilUnderlyingObjectError(d.Kind)
	}

	diff, err := jsondiff.Compare(d.DaemonSet, dsCopy.DaemonSet)
	if err != nil {
		return nil, fmt.Errorf("failed to compare daemonsets: %w", err)
	}

	return diff, nil
}
