package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getPrometheuses is the getResourceFunc for Prometheuses
func getPrometheuses(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	prometheuses, err := clientsets.Monitoring.MonitoringV1().Prometheuses(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	for _, item := range prometheuses.Items {
		results = append(results, &prometheus{item})
	}
	return results, nil
}

// prometheus is a wrapper for monitoring.coreos.com/v1.Prometheus to implement the Workload interface
type prometheus struct {
	*monitoringv1.Prometheus
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (p *prometheus) setReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < 0 {
		return errBoundOnScalingTargetValue
	}

	// #nosec G115
	newReplicas := int32(replicas)
	p.Spec.Replicas = &newReplicas
	return nil
}

// getCurrentReplicas gets the current amount of replicas of the resource
func (p *prometheus) getCurrentReplicas() (int, error) {
	replicas := p.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*p.Spec.Replicas), nil
}

// ScaleUp scales the resource up
func (p *prometheus) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(p)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
		return nil
	}

	err = p.setReplicas(*originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}
	removeOriginalReplicas(p)
	return nil
}

// ScaleDown scales the resource down
func (p *prometheus) ScaleDown(downscaleReplicas int) error {
	originalReplicas, err := p.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already scaled down, skipping", "workload", p.GetName(), "namespace", p.GetNamespace())
		return nil
	}

	err = p.setReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}
	setOriginalReplicas(originalReplicas, p)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (p *prometheus) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Monitoring.MonitoringV1().Prometheuses(p.Namespace).Update(ctx, p.Prometheus, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	return nil
}
