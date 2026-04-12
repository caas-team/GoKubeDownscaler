//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"
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
		results = append(results, &valueScaledWorkload{&service{&services.Items[i]}})
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
			results = append(results, &valueScaledWorkload{&service{svc}})
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
			results = append(results, &valueScaledWorkload{&service{svc}})
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

	return &valueScaledWorkload{&service{&svc}}, nil
}

// service is a wrapper for service.corev1 to implement the Workload interface.
type service struct {
	*corev1.Service
}

// setValue sets the value on the resource. Changes won't be made on Kubernetes until update() is called.
func (s *service) setValue(value values.Replicas) error {
	// only allow LoadBalancer or ClusterIP
	if value.String() != string(corev1.ServiceTypeLoadBalancer) && value.String() != string(corev1.ServiceTypeClusterIP) {
		allowed := fmt.Sprintf("%s or %s", string(corev1.ServiceTypeLoadBalancer), string(corev1.ServiceTypeClusterIP))
		return newUnexpectedOriginalReplicasError(allowed, value)
	}

	s.Spec.Type = corev1.ServiceType(value.String())

	return nil
}

// getValue gets the current value of the resource and the value used for downscaling,
//
//nolint:nonamedreturns //required to better understand the function
func (s *service) getValue() (currentValue, downscalingValue values.Replicas, err error) {
	currentValue = values.StatusReplicas(s.Spec.Type)
	downscalingValue = values.StatusReplicas(corev1.ServiceTypeClusterIP)

	return currentValue, downscalingValue, nil
}

// getSavedResourcesRequests gets the amount of resources that are requested to be saved by downscaling this resource.
func (s *service) getSavedResourcesRequests() *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
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

	return &valueScaledWorkload{&service{Service: copied}}, nil
}

// Compare compares two service resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (s *service) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	vsw, ok := workloadCopy.(*valueScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*valueScaledWorkload)(nil), workloadCopy)
	}

	svcCopy, ok := vsw.valueScaledResource.(*service)
	if !ok {
		return nil, newExpectTypeGotTypeError((*service)(nil), vsw.valueScaledResource)
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
