//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	downscalerIngressClassConst = "kube-downscaler-ingress-class"
)

// getIngress is the getResourceFunc for ingresses.
func getIngresses(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	ingresses, err := clientsets.Kubernetes.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ingresses: %w", err)
	}

	results := make([]Workload, 0, len(ingresses.Items))
	for i := range ingresses.Items {
		results = append(results, &valueScaledWorkload{&ingress{&ingresses.Items[i]}})
	}

	return results, nil
}

// parseIngressFromBytes parses the admission review and returns the ingress wrapped in a Workload.
func parseIngressesFromBytes(rawObject []byte) (Workload, error) {
	var ing networkingv1.Ingress
	if err := json.Unmarshal(rawObject, &ing); err != nil {
		return nil, fmt.Errorf("failed to decode Ingress: %w", err)
	}

	return &valueScaledWorkload{&ingress{&ing}}, nil
}

// ingress is a wrapper for ingress.networkingv1 to implement the Workload interface.
type ingress struct {
	*networkingv1.Ingress
}

// getSavedResourcesRequests gets the amount of resources that are requested to be saved by downscaling this resource.
func (i *ingress) getSavedResourcesRequests() *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// setValue sets the value on the resource. Changes won't be made on Kubernetes until update() is called.
func (i *ingress) setValue(targetReplicas values.Replicas) error {
	targetValue := targetReplicas.String()
	i.Spec.IngressClassName = &targetValue

	return nil
}

// getValue gets the current value of the resource and the value used for downscaling,
//
//nolint:nonamedreturns //required to better understand the function
func (i *ingress) getValue() (currentValue, downscalingValue values.Replicas, err error) {
	if i.Spec.IngressClassName == nil {
		return nil, nil, newIngressClassNameNilError(i.GetNamespace(), i.GetName())
	}

	currentValue = values.StatusReplicas(*i.Spec.IngressClassName)
	downscalingValue = values.StatusReplicas(downscalerIngressClassConst)

	return currentValue, downscalingValue, nil
}

// Reget regets the resource from the Kubernetes API.
func (i *ingress) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	i.Ingress, err = clientsets.Kubernetes.NetworkingV1().Ingresses(i.Namespace).Get(ctx, i.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ingress: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (i *ingress) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.NetworkingV1().Ingresses(i.Namespace).Update(ctx, i.Ingress, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ingress: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be an ingress.
func (i *ingress) Copy() (Workload, error) {
	if i.Ingress == nil {
		return nil, newNilUnderlyingObjectError(i.Kind)
	}

	copied := i.DeepCopy()

	return &valueScaledWorkload{&ingress{Ingress: copied}}, nil
}

// Compare compares two ingress resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (i *ingress) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	vsw, ok := workloadCopy.(*valueScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*valueScaledWorkload)(nil), workloadCopy)
	}

	ingCopy, ok := vsw.valueScaledResource.(*ingress)
	if !ok {
		return nil, newExpectTypeGotTypeError((*ingress)(nil), vsw.valueScaledResource)
	}

	if i.Ingress == nil || ingCopy.Ingress == nil {
		return nil, newNilUnderlyingObjectError(i.Kind)
	}

	diff, err := jsondiff.Compare(i.Ingress, ingCopy.Ingress)
	if err != nil {
		return nil, fmt.Errorf("failed to compare ingress: %w", err)
	}

	return diff, nil
}
