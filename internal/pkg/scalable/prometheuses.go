package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getPrometheuses is the getResourceFunc for Prometheuses.
func getPrometheuses(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	prometheuses, err := clientsets.Monitoring.MonitoringV1().Prometheuses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get prometheuses: %w", err)
	}

	results := make([]Workload, 0, len(prometheuses.Items))
	for i := range prometheuses.Items {
		results = append(results, &replicaScaledWorkload{&prometheus{&prometheuses.Items[i]}})
	}

	return results, nil
}

// parsePrometheusFromAdmissionRequest parses the admission review and returns the prometheus.
//
//nolint:ireturn //required for interface-based factory
func parsePrometheusFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var prom monitoringv1.Prometheus
	if err := json.Unmarshal(review.Request.Object.Raw, &prom); err != nil {
		return nil, fmt.Errorf("failed to decode Deployment: %w", err)
	}

	return &replicaScaledWorkload{&prometheus{&prom}}, nil
}

// prometheus is a wrapper for prometheus.v1.monitoring.coreos.com to implement the replicaScaledResource interface.
type prometheus struct {
	*monitoringv1.Prometheus
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (p *prometheus) setReplicas(replicas int32) error {
	p.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (p *prometheus) getReplicas() (values.Replicas, error) {
	replicas := p.Spec.Replicas
	if replicas == nil {
		return nil, newNoReplicasError(p.Kind, p.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (p *prometheus) Reget(clientsets *Clientsets, ctx context.Context) error {
	singlePrometheus, err := clientsets.Monitoring.MonitoringV1().Prometheuses(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get prometheus: %w", err)
	}

	p.Prometheus = singlePrometheus

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (p *prometheus) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Monitoring.MonitoringV1().Prometheuses(p.Namespace).Update(ctx, p.Prometheus, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update prometheus: %w", err)
	}

	return nil
}
