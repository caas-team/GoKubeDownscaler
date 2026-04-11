//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kruise "github.com/openkruise/kruise/apis/apps/v1alpha1"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getcloneSets is the getResourceFunc for cloneSets.
func getCloneSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	clonesets, err := clientsets.Kruise.AppsV1alpha1().CloneSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get clonesets: %w", err)
	}

	results := make([]Workload, 0, len(clonesets.Items))
	for i := range clonesets.Items {
		results = append(results, &replicaScaledWorkload{&cloneSet{&clonesets.Items[i]}})
	}

	return results, nil
}

// parsecloneSetFromBytes parses the admission review and returns the cloneset.
func parseCloneSetFromBytes(rawObject []byte) (Workload, error) {
	var cloneset kruise.CloneSet
	if err := json.Unmarshal(rawObject, &cloneset); err != nil {
		return nil, fmt.Errorf("failed to decode cloneset: %w", err)
	}

	return &replicaScaledWorkload{&cloneSet{&cloneset}}, nil
}

// cloneSet is a wrapper for cloneset.v1.apps to implement the replicaScaledResource interface.
type cloneSet struct {
	*kruise.CloneSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (c *cloneSet) setReplicas(replicas int32) error {
	c.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (c *cloneSet) getReplicas() (values.Replicas, error) {
	replicas := c.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(c.Kind, c.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (c *cloneSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	c.CloneSet, err = clientsets.Kruise.AppsV1alpha1().CloneSets(c.Namespace).Get(ctx, c.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cloneset: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the cloneSet.
func (c *cloneSet) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range c.Spec.Template.Spec.Containers {
		container := &c.Spec.Template.Spec.Containers[i]
		if container.Resources.Requests != nil {
			cpu := container.Resources.Requests.Cpu().AsApproximateFloat64()
			memory := container.Resources.Requests.Memory().AsApproximateFloat64()
			totalSavedCPU += cpu
			totalSavedMemory += memory
		}
	}

	totalSavedCPU *= float64(diffReplicas)
	totalSavedMemory *= float64(diffReplicas)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (c *cloneSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1alpha1().CloneSets(c.Namespace).Update(ctx, c.CloneSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cloneset: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a cloneSet.
func (c *cloneSet) Copy() (Workload, error) {
	if c.CloneSet == nil {
		return nil, newNilUnderlyingObjectError(c.Kind)
	}

	copied := c.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &cloneSet{
			CloneSet: copied,
		},
	}, nil
}

// Compare compares two cloneSet resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (c *cloneSet) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	cloneCopy, ok := rswCopy.replicaScaledResource.(*cloneSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*cloneSet)(nil), rswCopy.replicaScaledResource)
	}

	if c.CloneSet == nil || cloneCopy.CloneSet == nil {
		return nil, newNilUnderlyingObjectError(c.Kind)
	}

	diff, err := jsondiff.Compare(c.CloneSet, cloneCopy.CloneSet)
	if err != nil {
		return nil, fmt.Errorf("failed to compare clonesets: %w", err)
	}

	return diff, nil
}
