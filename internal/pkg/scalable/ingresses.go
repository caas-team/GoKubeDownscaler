package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	downscalerIngressClassConst = "downscaler-ingress-class"
)

// getIngress is the getResourceFunc for ingresses.
func getIngresses(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	ingresses, err := clientsets.Kubernetes.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ingresses: %w", err)
	}

	results := make([]Workload, 0, len(ingresses.Items))
	for i := range ingresses.Items {
		results = append(results, &ingress{&ingresses.Items[i]})
	}

	return results, nil
}

// parseIngressFromBytes parses the admission review and returns the ingress wrapped in a Workload.
func parseIngressesFromBytes(rawObject []byte) (Workload, error) {
	var ing networkingv1.Ingress
	if err := json.Unmarshal(rawObject, &ing); err != nil {
		return nil, fmt.Errorf("failed to decode Ingress: %w", err)
	}

	return &ingress{&ing}, nil
}

// ingress is a wrapper for ingress.networkingv1 to implement the Workload interface.
type ingress struct {
	*networkingv1.Ingress
}

// ScaleUp scales the resource up.
func (i *ingress) ScaleUp() error {
	originalState, err := getOriginalReplicas(i)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			slog.Debug("original replicas is not set, skipping", "workload", i.GetName(), "namespace", i.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	originalIngressClassName := originalState.String()
	i.Spec.IngressClassName = &originalIngressClassName

	removeOriginalReplicas(i)

	return nil
}

// ScaleDown scales the resource down.
func (i *ingress) ScaleDown(_ values.Replicas) (*metrics.SavedResources, error) {
	currentState := i.Spec.IngressClassName

	if *currentState == downscalerIngressClassConst {
		_, err := getOriginalReplicas(i)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if ok := errors.As(err, &originalReplicasUnsetErr); !ok {
				return metrics.NewSavedResources(0, 0), err
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", i.GetName(), "namespace", i.GetNamespace())

			return metrics.NewSavedResources(0, 0), nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", i.GetName(), "namespace", i.GetNamespace())

		return metrics.NewSavedResources(0, 0), nil
	}

	downscalerIngressClass := downscalerIngressClassConst
	i.Spec.IngressClassName = &downscalerIngressClass

	replicas := values.StatusReplicas(*currentState)
	setOriginalReplicas(replicas, i)

	return metrics.NewSavedResources(0, 0), nil
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

	return &ingress{Ingress: copied}, nil
}

// Compare compares two ingress resources and returns the differences as a jsondiff.Patch.
func (i *ingress) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	ingCopy, ok := workloadCopy.(*ingress)
	if !ok {
		return nil, newExpectTypeGotTypeError((*ingress)(nil), workloadCopy)
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
