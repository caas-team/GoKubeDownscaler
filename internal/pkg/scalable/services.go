package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AWSLoadBalancerAnnotation = "service.beta.kubernetes.io/aws-load-balancer-type"
)

// getServices is the getResourceFunc for services.
func getServices(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	services, err := clientsets.Kubernetes.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	results := make([]Workload, 0, len(services.Items))
	for i := range services.Items {
		results = append(results, &service{&services.Items[i]})
	}

	return results, nil
}

// getAWSELBServices is the getResourceFunc for AWS ELB services.
func getAWSELBServices(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	services, err := clientsets.Kubernetes.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	results := make([]Workload, 0, len(services.Items))
	for i := range services.Items {
		svc := &services.Items[i]

		if val, ok := svc.Annotations[AWSLoadBalancerAnnotation]; !ok || !strings.EqualFold(val, "nlb") {
			results = append(results, &service{svc})
		}
	}

	return results, nil
}

// getAWSNLBServices is the getResourceFunc for AWS NLB services.
func getAWSNLBServices(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	services, err := clientsets.Kubernetes.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	results := make([]Workload, 0, len(services.Items))
	for i := range services.Items {
		svc := &services.Items[i]

		if val, ok := svc.Annotations[AWSLoadBalancerAnnotation]; ok && strings.EqualFold(val, "nlb") {
			results = append(results, &service{svc})
		}
	}

	return results, nil
}

// parseServiceFromBytes parses the admission review and returns the service wrapped in a Workload.
func parseServiceFromBytes(rawObject []byte) (Workload, error) {
	var svc corev1.Service
	if err := json.Unmarshal(rawObject, &svc); err != nil {
		return nil, fmt.Errorf("failed to decode Service: %w", err)
	}

	return &service{&svc}, nil
}

// service is a wrapper for service.corev1 to implement the Workload interface.
type service struct {
	*corev1.Service
}

// ScaleUp scales the resource up.
func (s *service) ScaleUp() error {
	originalState, err := getOriginalReplicas(s)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			slog.Debug("original replicas is not set, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	if originalState.String() != string(corev1.ServiceTypeLoadBalancer) {
		return newUnexpectedOriginalReplicasError(string(corev1.ServiceTypeLoadBalancer), originalState.String())
	}

	s.Spec.Type = corev1.ServiceType(originalState.String())

	removeOriginalReplicas(s)

	return nil
}

// ScaleDown scales the resource down.
func (s *service) ScaleDown(_ values.Replicas) (*metrics.SavedResources, error) {
	currentState := string(s.Spec.Type)

	if currentState == string(corev1.ServiceTypeClusterIP) {
		_, err := getOriginalReplicas(s)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if ok := errors.As(err, &originalReplicasUnsetErr); !ok {
				return metrics.NewSavedResources(0, 0), err
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())

			return metrics.NewSavedResources(0, 0), nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())

		return metrics.NewSavedResources(0, 0), nil
	}

	s.Spec.Type = corev1.ServiceTypeClusterIP

	replicas := values.StatusReplicas(currentState)
	setOriginalReplicas(replicas, s)

	return metrics.NewSavedResources(0, 0), nil
}

// Reget regets the resource from the Kubernetes API.
func (s *service) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	s.Service, err = clientsets.Kubernetes.CoreV1().Services(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (s *service) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.CoreV1().Services(s.Namespace).Update(ctx, s.Service, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a service.
func (s *service) Copy() (Workload, error) {
	if s.Service == nil {
		return nil, newNilUnderlyingObjectError(s.Kind)
	}

	copied := s.DeepCopy()

	return &service{Service: copied}, nil
}

// Compare compares two service resources and returns the differences as a jsondiff.Patch.
func (s *service) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	svcCopy, ok := workloadCopy.(*service)
	if !ok {
		return nil, newExpectTypeGotTypeError((*service)(nil), workloadCopy)
	}

	if s.Service == nil || svcCopy.Service == nil {
		return nil, newNilUnderlyingObjectError(s.Kind)
	}

	diff, err := jsondiff.Compare(s.Service, svcCopy.Service)
	if err != nil {
		return nil, fmt.Errorf("failed to compare service: %w", err)
	}

	return diff, nil
}
