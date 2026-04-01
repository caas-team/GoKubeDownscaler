package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestIngress_ScaleDownAndUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		class                *string
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantClass            string
	}{
		{
			name:                 "scale down from nginx",
			class:                ptrString("nginx"),
			originalReplicas:     nil,
			wantOriginalReplicas: values.StatusReplicas("nginx"),
			wantClass:            downscalerIngressClassConst,
		},
		{
			name:                 "already downscaler without original state",
			class:                ptrString(downscalerIngressClassConst),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantClass:            downscalerIngressClassConst,
		},
		{
			name:                 "already downscaler but original state set",
			class:                ptrString(downscalerIngressClassConst),
			originalReplicas:     values.StatusReplicas("nginx"),
			wantOriginalReplicas: values.StatusReplicas("nginx"),
			wantClass:            downscalerIngressClassConst,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ing := &networkingv1.Ingress{}
			ing.Spec.IngressClassName = test.class

			workload := &ingress{ing}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			savedResources, err := workload.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)

			if test.wantClass != "" {
				if workload.Spec.IngressClassName == nil {
					t.Fatalf("expected ingress class %q but got nil", test.wantClass)
				}

				assert.Equal(t, test.wantClass, *workload.Spec.IngressClassName)
			}

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

	// ScaleUp specific tests
	t.Run("ingress scale up to original class", func(t *testing.T) {
		t.Parallel()

		ing := &networkingv1.Ingress{}
		ing.Spec.IngressClassName = ptrString(downscalerIngressClassConst)

		workload := &ingress{ing}
		setOriginalReplicas(values.StatusReplicas("nginx"), workload)

		err := workload.ScaleUp()
		require.NoError(t, err)

		if workload.Spec.IngressClassName == nil {
			t.Fatalf("expected ingress class set after scale up")
		}

		assert.Equal(t, "nginx", *workload.Spec.IngressClassName)

		// annotation should be removed
		_, err = getOriginalReplicas(workload)
		var unsetErr *OriginalReplicasUnsetError
		assert.ErrorAs(t, err, &unsetErr)
	})

	t.Run("ingress scale up when original not set should be noop", func(t *testing.T) {
		t.Parallel()

		ing := &networkingv1.Ingress{}
		ing.Spec.IngressClassName = ptrString("nginx")

		workload := &ingress{ing}

		err := workload.ScaleUp()
		require.NoError(t, err)

		assert.Equal(t, "nginx", *workload.Spec.IngressClassName)
	})
}
