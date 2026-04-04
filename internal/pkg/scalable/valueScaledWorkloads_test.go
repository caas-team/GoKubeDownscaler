package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	IngressKind = "ingress"
	ServiceKind = "service"
	GatewayKind = "gateway"
)

// buildValueScaledResourceForTest constructs a valueScaledResource and returns the underlying concrete object for assertions in tests.
//
//nolint:nonamedreturns // the named return values make it easier to understand the purpose of the returned values in the test code.
func buildValueScaledResourceForTest(t *testing.T, kind string, initial any) (valueScaledResource valueScaledResource, obj any) {
	t.Helper()

	switch kind {
	case ServiceKind:
		svc := &corev1.Service{}
		svc.Spec.Type = initial.(corev1.ServiceType)

		return &service{svc}, svc
	case IngressKind:
		class := initial.(string)
		ing := &networkingv1.Ingress{}
		ing.Spec.IngressClassName = &class

		return &ingress{ing}, ing
	case GatewayKind:
		gtw := &gatewayv1.Gateway{}
		gtw.Spec.GatewayClassName = initial.(gatewayv1.ObjectName)

		return &gateway{gtw}, gtw
	default:
		t.Fatalf("unknown kind %q", kind)
		return nil, nil
	}
}

// Table-driven tests for valueScaledWorkload ScaleDown behavior for services, ingresses and gateways.
func TestValueScaledWorkload_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		kind                 string
		initial              any
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantTargetState      string
	}{
		{
			name:                 "service scale down from LoadBalancer",
			kind:                 ServiceKind,
			initial:              corev1.ServiceTypeLoadBalancer,
			originalReplicas:     nil,
			wantOriginalReplicas: values.StatusReplicas(corev1.ServiceTypeLoadBalancer),
			wantTargetState:      string(corev1.ServiceTypeClusterIP),
		},
		{
			name:                 "ingress scale down from nginx",
			kind:                 IngressKind,
			initial:              "nginx",
			originalReplicas:     nil,
			wantOriginalReplicas: values.StatusReplicas("nginx"),
			wantTargetState:      downscalerIngressClassConst,
		},
		{
			name:                 "gateway scale down from nginx",
			kind:                 GatewayKind,
			initial:              gatewayv1.ObjectName("nginx"),
			originalReplicas:     nil,
			wantOriginalReplicas: values.StatusReplicas("nginx"),
			wantTargetState:      downscalerGatewayClassConst,
		},
		{
			name:                 "service already at target ClusterIP",
			kind:                 ServiceKind,
			initial:              corev1.ServiceTypeClusterIP,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantTargetState:      string(corev1.ServiceTypeClusterIP),
		},
		{
			name:                 "ingress already at target class",
			kind:                 IngressKind,
			initial:              downscalerIngressClassConst,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantTargetState:      downscalerIngressClassConst,
		},
		{
			name:                 "gateway already at target class",
			kind:                 GatewayKind,
			initial:              gatewayv1.ObjectName(downscalerGatewayClassConst),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantTargetState:      downscalerGatewayClassConst,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w, underlying := buildValueScaledResourceForTest(t, test.kind, test.initial)

			vsw := &valueScaledWorkload{w}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, vsw)
			}

			saved, err := vsw.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)
			require.NotNil(t, saved)

			switch test.kind {
			case ServiceKind:
				svc := underlying.(*corev1.Service)
				assert.Equal(t, test.wantTargetState, string(svc.Spec.Type))
			case IngressKind:
				ing := underlying.(*networkingv1.Ingress)
				if ing.Spec.IngressClassName == nil {
					t.Fatalf("expected ingress class set")
				}

				assert.Equal(t, test.wantTargetState, *ing.Spec.IngressClassName)
			case GatewayKind:
				gtw := underlying.(*gatewayv1.Gateway)
				assert.Equal(t, test.wantTargetState, string(gtw.Spec.GatewayClassName))
			}

			gotOriginal, err := getOriginalReplicas(vsw)

			var unsetErr *OriginalReplicasUnsetError
			if !errors.As(err, &unsetErr) {
				require.NoError(t, err)
			}

			assert.Equal(t, test.wantOriginalReplicas, gotOriginal)
		})
	}
}

// Table-driven tests for ScaleUp behavior.
func TestValueScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		kind             string
		initialDownState any
		originalReplicas values.Replicas
		wantRestored     string
	}{
		{
			name:             "service scale up restores original type",
			kind:             ServiceKind,
			initialDownState: corev1.ServiceTypeClusterIP,
			originalReplicas: values.StatusReplicas(corev1.ServiceTypeLoadBalancer),
			wantRestored:     string(corev1.ServiceTypeLoadBalancer),
		},
		{
			name:             "ingress scale up restores original class",
			kind:             IngressKind,
			initialDownState: downscalerIngressClassConst,
			originalReplicas: values.StatusReplicas("nginx"),
			wantRestored:     "nginx",
		},
		{
			name:             "gateway scale up restores original class",
			kind:             GatewayKind,
			initialDownState: gatewayv1.ObjectName(downscalerGatewayClassConst),
			originalReplicas: values.StatusReplicas("nginx"),
			wantRestored:     "nginx",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w, underlying := buildValueScaledResourceForTest(t, test.kind, test.initialDownState)

			vsw := &valueScaledWorkload{w}
			setOriginalReplicas(test.originalReplicas, vsw)

			err := vsw.ScaleUp()
			require.NoError(t, err)

			switch test.kind {
			case ServiceKind:
				svc := underlying.(*corev1.Service)
				assert.Equal(t, test.wantRestored, string(svc.Spec.Type))
			case IngressKind:
				ing := underlying.(*networkingv1.Ingress)
				if ing.Spec.IngressClassName == nil {
					t.Fatalf("expected ingress class set")
				}

				assert.Equal(t, test.wantRestored, *ing.Spec.IngressClassName)
			case GatewayKind:
				gtw := underlying.(*gatewayv1.Gateway)
				assert.Equal(t, test.wantRestored, string(gtw.Spec.GatewayClassName))
			}

			// ensure original annotation removed
			_, err = getOriginalReplicas(vsw)
			var unsetErr *OriginalReplicasUnsetError
			assert.ErrorAs(t, err, &unsetErr)
		})
	}
}
