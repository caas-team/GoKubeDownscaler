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

// parseDaemonSetFromAdmissionRequest parses the admission review and returns the daemonset.
func parseDaemonSetFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var ds appsv1.DaemonSet
	if err := json.Unmarshal(review.Request.Object.Raw, &ds); err != nil {
		return nil, fmt.Errorf("failed to decode daemonset: %v", err)
	}
	return &daemonSet{&ds}, nil
}

// daemonSet is a wrapper for daemonset.v1.apps to implement the Workload interface.
type daemonSet struct {
	*appsv1.DaemonSet
}

// ScaleUp scales the resource up.
func (d *daemonSet) ScaleUp() error {
	delete(d.Spec.Template.Spec.NodeSelector, labelMatchNone)
	return nil
}

// ScaleDown scales the resource down.
func (d *daemonSet) ScaleDown(_ values.Replicas) error {
	if d.Spec.Template.Spec.NodeSelector == nil {
		d.Spec.Template.Spec.NodeSelector = map[string]string{}
	}

	d.Spec.Template.Spec.NodeSelector[labelMatchNone] = "true"

	return nil
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

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (d *daemonSet) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AppsV1().DaemonSets(d.Namespace).Update(ctx, d.DaemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset: %w", err)
	}

	return nil
}
