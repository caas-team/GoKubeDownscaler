package metrics

import (
	k8smetrics "k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
	_ "k8s.io/component-base/metrics/prometheus/restclient"
)

const (
	validateWorkloadPath = "/validate-workload"
	mutatedLabel         = "mutated"
	notMutatedLabel      = "not_mutated"
	errorLabel           = "error"
)

type AdmissionMetrics struct {
	validateWorkloadAdmissionRequestsTotal *k8smetrics.CounterVec
	admissionRequestDurationSeconds        *k8smetrics.HistogramVec
}

func NewAdmissionMetrics(dryRun bool) *AdmissionMetrics {
	return &AdmissionMetrics{
		validateWorkloadAdmissionRequestsTotal: k8smetrics.NewCounterVec(
			&k8smetrics.CounterOpts{
				Name: metricName("validate_workload_admission_requests_total", dryRun),
				Help: helperDescription("total number of validateWorkloadAdmissionRequests, broken down by namespace and result of mutation"+
					" (mutated, not_mutated, error).", dryRun),
			}, []string{"namespace", "result"},
		),
		admissionRequestDurationSeconds: k8smetrics.NewHistogramVec(
			&k8smetrics.HistogramOpts{
				Name: metricName("admission_request_duration_seconds", dryRun),
				Help: helperDescription("Duration of admission webhook requests in seconds, broken down by request path.", dryRun),
				Buckets: []float64{
					0.025, 0.05, 0.1, 0.25,
					0.5, 1.0, 2.5, 5.0,
				},
			}, []string{"path"},
		),
	}
}

func (m *AdmissionMetrics) RegisterAll() {
	legacyregistry.MustRegister(m.validateWorkloadAdmissionRequestsTotal)
	legacyregistry.MustRegister(m.admissionRequestDurationSeconds)
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

func (m *AdmissionMetrics) UpdateAdmissionRequestDurationSecondsHistogram(
	metricsEnabled bool,
	cycleDuration float64,
) {
	if !metricsEnabled {
		return
	}

	m.admissionRequestDurationSeconds.WithLabelValues(validateWorkloadPath).Observe(cycleDuration)
}
