//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	downscalerGatewayClassConst = "kube-downscaler-gateway-class"
)

// getGateways is the getResourceFunc for gateways.
func getGateways(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	gateways, err := clientsets.Gateway.GatewayV1().Gateways(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get gateways: %w", err)
	}

	results := make([]Workload, 0, len(gateways.Items))
	for i := range gateways.Items {
		results = append(results, &valueScaledWorkload{&gateway{&gateways.Items[i]}})
	}

	return results, nil
}

// parseIngressFromBytes parses the admission review and returns the gateway wrapped in a Workload.
func parseGatewaysFromBytes(rawObject []byte) (Workload, error) {
	var gtw gatewayv1.Gateway
	if err := json.Unmarshal(rawObject, &gtw); err != nil {
		return nil, fmt.Errorf("failed to decode Gateway: %w", err)
	}

	return &valueScaledWorkload{&gateway{&gtw}}, nil
}

// ingress is a wrapper for ingress.networkingv1 to implement the Workload interface.
type gateway struct {
	*gatewayv1.Gateway
}

// setValue sets the value on the resource. Changes won't be made on Kubernetes until update() is called.
func (g *gateway) setValue(targetReplicas values.Replicas) error {
	g.Spec.GatewayClassName = gatewayv1.ObjectName(targetReplicas.String())

	return nil
}

// getValue gets the current value of the resource and the value used for downscaling,
//
//nolint:nonamedreturns //required to better understand the function
func (g *gateway) getValue() (currentValue, downscalingValue values.Replicas, err error) {
	currentValue = values.StatusReplicas(g.Spec.GatewayClassName)
	downscalingValue = values.StatusReplicas(downscalerGatewayClassConst)

	return currentValue, downscalingValue, nil
}

// getSavedResourcesRequests gets the amount of resources that are requested to be saved by downscaling this resource.
func (g *gateway) getSavedResourcesRequests() *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
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

	return &valueScaledWorkload{valueScaledResource: &gateway{Gateway: copied}}, nil
}

// Compare compares two ingress resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (g *gateway) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	vswCopy, ok := workloadCopy.(*valueScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*valueScaledWorkload)(nil), workloadCopy)
	}

	gtwCopy, ok := vswCopy.valueScaledResource.(*gateway)
	if !ok {
		return nil, newExpectTypeGotTypeError((*gateway)(nil), vswCopy.valueScaledResource)
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
