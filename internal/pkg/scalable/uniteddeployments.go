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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStatefulSets is the getResourceFunc for uniteddeployments.
func getUnitedDeployments(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	uniteddeployments, err := clientsets.Kruise.AppsV1alpha1().UnitedDeployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get uniteddeployments: %w", err)
	}

	results := make([]Workload, 0, len(uniteddeployments.Items))
	for i := range uniteddeployments.Items {
		results = append(results, &replicaScaledWorkload{&unitedDeployment{&uniteddeployments.Items[i]}})
	}

	return results, nil
}

// parseStatefulSetFromBytes parses the admission review and returns the unitedeployment.
func parseUnitedDeploymentsFromBytes(rawObject []byte) (Workload, error) {
	var udeploy kruise.UnitedDeployment
	if err := json.Unmarshal(rawObject, &udeploy); err != nil {
		return nil, fmt.Errorf("failed to decode UnitedDeployment: %w", err)
	}

	return &replicaScaledWorkload{&unitedDeployment{&udeploy}}, nil
}

// advancedStatefulSet is a wrapper for unitedeployment.v1.apps to implement the replicaScaledResource interface.
type unitedDeployment struct {
	*kruise.UnitedDeployment
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (u *unitedDeployment) setReplicas(replicas int32) error {
	u.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (u *unitedDeployment) getReplicas() (values.Replicas, error) {
	replicas := u.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(u.Kind, u.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (u *unitedDeployment) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	u.UnitedDeployment, err = clientsets.Kruise.AppsV1alpha1().UnitedDeployments(u.Namespace).Get(ctx, u.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get uniteddeployment: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the StatefulSet.
func (u *unitedDeployment) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	var containers []corev1.Container

	switch {
	case u.Spec.Template.DeploymentTemplate != nil:
		containers = u.Spec.Template.DeploymentTemplate.Spec.Template.Spec.Containers
	case u.Spec.Template.CloneSetTemplate != nil:
		containers = u.Spec.Template.CloneSetTemplate.Spec.Template.Spec.Containers
	case u.Spec.Template.StatefulSetTemplate != nil:
		containers = u.Spec.Template.StatefulSetTemplate.Spec.Template.Spec.Containers
	case u.Spec.Template.AdvancedStatefulSetTemplate != nil:
		containers = u.Spec.Template.AdvancedStatefulSetTemplate.Spec.Template.Spec.Containers
	}

	for i := range containers {
		container := &containers[i]
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
func (u *unitedDeployment) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kruise.AppsV1alpha1().UnitedDeployments(u.Namespace).Update(ctx, u.UnitedDeployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update uniteddeployment: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a statefulSet.
func (u *unitedDeployment) Copy() (Workload, error) {
	if u.UnitedDeployment == nil {
		return nil, newNilUnderlyingObjectError(u.Kind)
	}

	copied := u.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &unitedDeployment{
			UnitedDeployment: copied,
		},
	}, nil
}

// Compare compares two statefulSet resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (u *unitedDeployment) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	unitedCopy, ok := rswCopy.replicaScaledResource.(*unitedDeployment)
	if !ok {
		return nil, newExpectTypeGotTypeError((*unitedDeployment)(nil), rswCopy.replicaScaledResource)
	}

	if u.UnitedDeployment == nil || unitedCopy.UnitedDeployment == nil {
		return nil, newNilUnderlyingObjectError(u.Kind)
	}

	diff, err := jsondiff.Compare(u.UnitedDeployment, unitedCopy.UnitedDeployment)
	if err != nil {
		return nil, fmt.Errorf("failed to compare uniteddeployments: %w", err)
	}

	return diff, nil
}
