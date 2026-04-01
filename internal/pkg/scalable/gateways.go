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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	downscalerGatewayClassConst = "downscaler-gateway-class"
)

// getGateways is the getResourceFunc for gateways.
func getGateways(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	gateways, err := clientsets.Gateway.GatewayV1().Gateways(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get gateways: %w", err)
	}

	results := make([]Workload, 0, len(gateways.Items))
	for i := range gateways.Items {
		results = append(results, &gateway{&gateways.Items[i]})
	}

	return results, nil
}

// parseIngressFromBytes parses the admission review and returns the gateway wrapped in a Workload.
func parseGatewaysFromBytes(rawObject []byte) (Workload, error) {
	var gtw gatewayv1.Gateway
	if err := json.Unmarshal(rawObject, &gtw); err != nil {
		return nil, fmt.Errorf("failed to decode Gateway: %w", err)
	}

	return &gateway{&gtw}, nil
}

// ingress is a wrapper for ingress.networkingv1 to implement the Workload interface.
type gateway struct {
	*gatewayv1.Gateway
}

// ScaleUp scales the resource up.
func (g *gateway) ScaleUp() error {
	originalState, err := getOriginalReplicas(g)
	if err != nil {
		var originalReplicasUnsetError *OriginalReplicasUnsetError
		if ok := errors.As(err, &originalReplicasUnsetError); ok {
			slog.Debug("original replicas is not set, skipping", "workload", g.GetName(), "namespace", g.GetNamespace())
			return nil
		}

		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}

	g.Spec.GatewayClassName = gatewayv1.ObjectName(originalState.String())

	removeOriginalReplicas(g)

	return nil
}

// ScaleDown scales the resource down.
func (g *gateway) ScaleDown(_ values.Replicas) (*metrics.SavedResources, error) {
	currentState := g.Spec.GatewayClassName

	if currentState == downscalerGatewayClassConst {
		_, err := getOriginalReplicas(g)

		var originalReplicasUnsetErr *OriginalReplicasUnsetError
		if err != nil {
			if ok := errors.As(err, &originalReplicasUnsetErr); !ok {
				return metrics.NewSavedResources(0, 0), err
			}

			slog.Debug("workload is already at target scale down state, skipping", "workload", g.GetName(), "namespace", g.GetNamespace())

			return metrics.NewSavedResources(0, 0), nil
		}

		slog.Debug("workload is already scaled down, skipping", "workload", g.GetName(), "namespace", g.GetNamespace())

		return metrics.NewSavedResources(0, 0), nil
	}

	g.Spec.GatewayClassName = downscalerGatewayClassConst

	replicas := values.StatusReplicas(currentState)
	setOriginalReplicas(replicas, g)

	return metrics.NewSavedResources(0, 0), nil
}

// Reget regets the resource from the Kubernetes API.
func (g *gateway) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	g.Gateway, err = clientsets.Gateway.GatewayV1().Gateways(g.Namespace).Get(ctx, g.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get gateway: %w", err)
	}

	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (g *gateway) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Gateway.GatewayV1().Gateways(g.Namespace).Update(ctx, g.Gateway, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update gateway: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a gateway.
func (g *gateway) Copy() (Workload, error) {
	if g.Gateway == nil {
		return nil, newNilUnderlyingObjectError(g.Kind)
	}

	copied := g.DeepCopy()

	return &gateway{Gateway: copied}, nil
}

// Compare compares two ingress resources and returns the differences as a jsondiff.Patch.
func (g *gateway) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	gtwCopy, ok := workloadCopy.(*gateway)
	if !ok {
		return nil, newExpectTypeGotTypeError((*gateway)(nil), workloadCopy)
	}

	if g.Gateway == nil || gtwCopy.Gateway == nil {
		return nil, newNilUnderlyingObjectError(g.Kind)
	}

	diff, err := jsondiff.Compare(g.Gateway, gtwCopy.Gateway)
	if err != nil {
		return nil, fmt.Errorf("failed to compare gateway: %w", err)
	}

	return diff, nil
}
