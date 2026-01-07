package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	_ "k8s.io/component-base/metrics/prometheus/restclient"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	mutatedLabel    = "mutated"
	notMutatedLabel = "not_mutated"
	errorLabel      = "error"
)

type AdmissionMetrics struct {
	validateWorkloadAdmissionRequestsTotal *prometheus.CounterVec
}

func NewAdmissionMetrics(dryRun bool) *AdmissionMetrics {
	return &AdmissionMetrics{
		validateWorkloadAdmissionRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricName("validate_workload_admission_requests_total", dryRun),
				Help: helperDescription("total number of validateWorkloadAdmissionRequests, broken down by namespace and result of mutation"+
					" (mutated, not_mutated, error).", dryRun),
			}, []string{"namespace", "result"},
		),
	}
}

func (m *AdmissionMetrics) RegisterAll() {
	metrics.Registry.MustRegister(m.validateWorkloadAdmissionRequestsTotal)
}

func (m *AdmissionMetrics) UpdateValidateWorkloadAdmissionRequestsTotal(
	metricsEnabled bool,
	mutated bool,
	err bool,
	namespace string,
) {
	if !metricsEnabled {
		return
	}

	if m != nil && err {
		m.validateWorkloadAdmissionRequestsTotal.WithLabelValues(namespace, errorLabel).Inc()
	}

	if m != nil && mutated {
		m.validateWorkloadAdmissionRequestsTotal.WithLabelValues(namespace, mutatedLabel).Inc()
	}

	if m != nil && !mutated {
		m.validateWorkloadAdmissionRequestsTotal.WithLabelValues(namespace, notMutatedLabel).Inc()
	}
}

func (m *AdmissionMetrics) DeleteNamespaceMetrics(namespace string) {
	if m != nil {
		m.validateWorkloadAdmissionRequestsTotal.DeletePartialMatch(prometheus.Labels{"namespace": namespace})
	}
}
