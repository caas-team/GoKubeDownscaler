package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGateway_ScaleDownAndUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		class                gatewayv1.ObjectName
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantClass            gatewayv1.ObjectName
	}{
		{
			name:                 "scale down from nginx",
			class:                gatewayv1.ObjectName("nginx"),
			originalReplicas:     nil,
			wantOriginalReplicas: values.StatusReplicas("nginx"),
			wantClass:            gatewayv1.ObjectName(downscalerGatewayClassConst),
		},
		{
			name:                 "already downscaler without original state",
			class:                gatewayv1.ObjectName(downscalerGatewayClassConst),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantClass:            gatewayv1.ObjectName(downscalerGatewayClassConst),
		},
		{
			name:                 "already downscaler but original state set",
			class:                gatewayv1.ObjectName(downscalerGatewayClassConst),
			originalReplicas:     values.StatusReplicas("nginx"),
			wantOriginalReplicas: values.StatusReplicas("nginx"),
			wantClass:            gatewayv1.ObjectName(downscalerGatewayClassConst),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gtw := &gatewayv1.Gateway{}
			gtw.Spec.GatewayClassName = test.class

			workload := &gateway{gtw}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			savedResources, err := workload.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)

			assert.Equal(t, string(test.wantClass), string(workload.Spec.GatewayClassName))

			gotOriginal, err := getOriginalReplicas(workload)
			var unsetErr *OriginalReplicasUnsetError

			if !errors.As(err, &unsetErr) {
				require.NoError(t, err)
			}

			assert.Equal(t, test.wantOriginalReplicas, gotOriginal)

			assert.InDelta(t, 0.0, savedResources.TotalCPU(), 0.0001)
			assert.InDelta(t, 0.0, savedResources.TotalMemory(), 1e5)
		})
	}

	t.Run("gateway scale up to original class", func(t *testing.T) {
		t.Parallel()

		gtw := &gatewayv1.Gateway{}
		gtw.Spec.GatewayClassName = gatewayv1.ObjectName(downscalerGatewayClassConst)

		workload := &gateway{gtw}
		setOriginalReplicas(values.StatusReplicas("nginx"), workload)

		err := workload.ScaleUp()
		require.NoError(t, err)

		assert.Equal(t, "nginx", string(workload.Spec.GatewayClassName))

		// annotation should be removed
		_, err = getOriginalReplicas(workload)
		var unsetErr *OriginalReplicasUnsetError
		assert.ErrorAs(t, err, &unsetErr)
	})

	t.Run("gateway scale up when original not set should be noop", func(t *testing.T) {
		t.Parallel()

		gtw := &gatewayv1.Gateway{}
		gtw.Spec.GatewayClassName = gatewayv1.ObjectName("nginx")

		workload := &gateway{gtw}

		err := workload.ScaleUp()
		require.NoError(t, err)

		assert.Equal(t, "nginx", string(workload.Spec.GatewayClassName))
	})
}
