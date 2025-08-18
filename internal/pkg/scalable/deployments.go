//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DeploymentKind = "Deployment"

// getDeployments is the getResourceFunc for Deployments.
func getDeployments(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	deployments, err := clientsets.Kubernetes.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	results := make([]Workload, 0, len(deployments.Items))
	for i := range deployments.Items {
		results = append(results, &replicaScaledWorkload{&deployment{&deployments.Items[i]}})
	}

	return results, nil
}

// parseDeploymentFromAdmissionRequest parses the admission review and returns the deployment.
//
//nolint:ireturn // required for interface-based factory
func parseDeploymentFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var dep appsv1.Deployment
	if err := json.Unmarshal(review.Request.Object.Raw, &dep); err != nil {
		return nil, fmt.Errorf("failed to decode deployment: %w", err)
	}

	return &replicaScaledWorkload{&deployment{&dep}}, nil
}

// deployment is a wrapper for deployment.v1.apps to implement the replicaScaledResource interface.
type deployment struct {
	*appsv1.Deployment
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (d *deployment) setReplicas(replicas int32) error {
	d.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (d *deployment) getReplicas() (values.Replicas, error) {
	replicas := d.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(d.Kind, d.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (d *deployment) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	d.Deployment, err = clientsets.Kubernetes.AppsV1().Deployments(d.Namespace).Get(ctx, d.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cronjob: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (d *deployment) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().Deployments(d.Namespace).Update(ctx, d.Deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a deployment.
//
//nolint:ireturn //required for interface-based workflow
func (d *deployment) Copy() (Workload, error) {
	if d.Deployment == nil {
		return nil, newNilUnderlyingObjectError(DeploymentKind)
	}

	copied := d.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &deployment{
			Deployment: copied,
		},
	}, nil
}

// Compare compares two deployment resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (d *deployment) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	depCopy, ok := rswCopy.replicaScaledResource.(*deployment)
	if !ok {
		return nil, newExpectTypeGotTypeError((*deployment)(nil), rswCopy.replicaScaledResource)
	}

	if d.Deployment == nil || depCopy.Deployment == nil {
		return nil, newNilUnderlyingObjectError(DeploymentKind)
	}

	diff, err := jsondiff.Compare(d.Deployment, depCopy.Deployment)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError(DeploymentKind, err)
	}

	return diff, nil
}
