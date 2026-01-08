//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/wI2L/jsondiff"
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

// parsePrometheusFromBytes parses the admission review and returns the prometheus.
func parsePrometheusFromBytes(rawObject []byte) (Workload, error) {
	var prom monitoringv1.Prometheus
	if err := json.Unmarshal(rawObject, &prom); err != nil {
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

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the Prometheus.
//

func (p *prometheus) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range p.Spec.Containers {
		container := &p.Spec.Containers[i] // take pointer to avoid copying
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	totalSavedCPU *= float64(diffReplicas)
	totalSavedMemory *= float64(diffReplicas)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (p *prometheus) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Monitoring.MonitoringV1().Prometheuses(p.Namespace).Update(ctx, p.Prometheus, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update prometheus: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a prometheus.
func (p *prometheus) Copy() (Workload, error) {
	if p.Prometheus == nil {
		return nil, newNilUnderlyingObjectError(p.Kind)
	}

	copied := p.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &prometheus{
			Prometheus: copied,
		},
	}, nil
}

// Compare compares two prometheus resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (p *prometheus) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	promCopy, ok := rswCopy.replicaScaledResource.(*prometheus)
	if !ok {
		return nil, newExpectTypeGotTypeError((*prometheus)(nil), rswCopy.replicaScaledResource)
	}

	if p.Prometheus == nil || promCopy.Prometheus == nil {
		return nil, newNilUnderlyingObjectError(p.Kind)
	}

	diff, err := jsondiff.Compare(p.Prometheus, promCopy.Prometheus)
	if err != nil {
		return nil, fmt.Errorf("failed to compare prometheus: %w", err)
	}

	return diff, nil
}
