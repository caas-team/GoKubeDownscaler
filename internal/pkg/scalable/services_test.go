package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestService_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		serviceType          corev1.ServiceType
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantType             corev1.ServiceType
		wantSavedCPU         float64
		wantSavedMemory      float64
	}{
		{
			name:                 "scale down from LoadBalancer",
			serviceType:          corev1.ServiceTypeLoadBalancer,
			originalReplicas:     nil,
			wantOriginalReplicas: values.StatusReplicas(corev1.ServiceTypeLoadBalancer),
			wantType:             corev1.ServiceTypeClusterIP,
			wantSavedCPU:         0,
			wantSavedMemory:      0,
		},
		{
			name:                 "already clusterIP without original state",
			serviceType:          corev1.ServiceTypeClusterIP,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantType:             corev1.ServiceTypeClusterIP,
			wantSavedCPU:         0,
			wantSavedMemory:      0,
		},
		{
			name:                 "already clusterIP but original state set",
			serviceType:          corev1.ServiceTypeClusterIP,
			originalReplicas:     values.StatusReplicas(corev1.ServiceTypeLoadBalancer),
			wantOriginalReplicas: values.StatusReplicas(corev1.ServiceTypeLoadBalancer),
			wantType:             corev1.ServiceTypeClusterIP,
			wantSavedCPU:         0,
			wantSavedMemory:      0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			svc := &corev1.Service{}
			svc.Spec.Type = test.serviceType

			workload := &service{svc}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			savedResources, err := workload.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)

			assert.Equal(t, test.wantType, svc.Spec.Type)

			gotOriginal, err := getOriginalReplicas(workload)
			var unsetErr *OriginalReplicasUnsetError

			if !errors.As(err, &unsetErr) {
				require.NoError(t, err)
			}

			assert.Equal(t, test.wantOriginalReplicas, gotOriginal)

			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5)
		})
	}
}

func TestService_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		serviceType      corev1.ServiceType
		originalReplicas values.Replicas
		wantType         corev1.ServiceType
		wantErr          bool
	}{
		{
			name:             "scale up to original type",
			serviceType:      corev1.ServiceTypeClusterIP,
			originalReplicas: values.StatusReplicas(corev1.ServiceTypeLoadBalancer),
			wantType:         corev1.ServiceTypeLoadBalancer,
		},
		{
			name:             "original replicas not set",
			serviceType:      corev1.ServiceTypeClusterIP,
			originalReplicas: nil,
			wantType:         corev1.ServiceTypeClusterIP,
		},
		{
			name:             "original replicas invalid type",
			serviceType:      corev1.ServiceTypeClusterIP,
			originalReplicas: values.AbsoluteReplicas(5),
			wantErr:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			svc := &corev1.Service{}
			svc.Spec.Type = test.serviceType

			workload := &service{svc}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			err := workload.ScaleUp()

			if test.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, test.wantType, svc.Spec.Type)
		})
	}
}
