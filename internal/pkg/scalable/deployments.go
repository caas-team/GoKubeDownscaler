//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
