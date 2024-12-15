package scalable

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getPrometheuses is the getResourcesFunc for Prometheuses
func getPrometheuses(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	prometheuses, err := clientsets.Monitoring.MonitoringV1().Prometheuses(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get prometheuses: %w", err)
	}
	for _, item := range prometheuses.Items {
		results = append(results, &replicaScaledWorkload{&prometheus{item}})
	}
	return results, nil
}

// getPrometheus is the getResourceFunc for Prometheus
func getPrometheus(name string, namespace string, clientsets *Clientsets, ctx context.Context) (Workload, error) {
	var result Workload
	prom, err := clientsets.Monitoring.MonitoringV1().Prometheuses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get prometheus: %w", err)
	}
	result = &replicaScaledWorkload{&prometheus{prom}}
	return result, nil
}

// prometheus is a wrapper for monitoring.coreos.com/v1.Prometheus to implement the replicaScaledResource interface
type prometheus struct {
	*monitoringv1.Prometheus
}

// GetResourceType returns the name of the workload type
func (p *prometheus) GetResourceType() string {
	return "prometheus"
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (p *prometheus) setReplicas(replicas int32) error {
	p.Spec.Replicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource
func (p *prometheus) getReplicas() (int32, error) {
	replicas := p.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return *p.Spec.Replicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (p *prometheus) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Monitoring.MonitoringV1().Prometheuses(p.Namespace).Update(ctx, p.Prometheus, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update prometheus: %w", err)
	}
	return nil
}
