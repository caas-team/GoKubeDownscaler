package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestReplicaScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		replicas             values.Replicas
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantReplicas         values.Replicas
		wantErr              error
	}{
		{
			name:                 "scale up",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
		},
		{
			name:                 "already scaled up",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
		},
		{
			name:                 "original replicas not set",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "original replicas is not AbsoluteReplicas",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.PercentageReplicas(50),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(0),
			wantErr:              &values.InvalidReplicaTypeError{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deployment := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
			replicasInt32, _ := test.replicas.AsInt32()
			_ = deployment.setReplicas(replicasInt32)

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, deployment)
			}

			err := deployment.ScaleUp()
			var invalidReplicaTypeError *values.InvalidReplicaTypeError

			if errors.As(test.wantErr, &invalidReplicaTypeError) {
				assert.ErrorAs(t, err, &invalidReplicaTypeError)
				return
			}

			require.NoError(t, err)

			replicas, err := deployment.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, replicas)

			oringalReplicas, err := getOriginalReplicas(deployment)
			var unsetErr *OriginalReplicasUnsetError

			if !errors.As(err, &unsetErr) {
				require.NoError(t, err)
			}

			assert.Equal(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestReplicaScaledWorkload_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		replicas             values.Replicas
		originalReplicas     values.Replicas
		downtimeReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantReplicas         values.Replicas
		wantSavedCPU         float64 // in cores
		wantSavedMemory      float64 // in bytes
		wantErr              error
	}{
		{
			name:                 "scale down",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			downtimeReplicas:     values.AbsoluteReplicas(0),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         values.AbsoluteReplicas(0),
			wantSavedCPU:         0.5,               // 5 replicas × 0.1 cores each
			wantSavedMemory:      320 * 1024 * 1024, // 5 replicas × 64Mi = 320MiB
		},
		{
			name:                 "already scaled down",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.AbsoluteReplicas(5),
			downtimeReplicas:     values.AbsoluteReplicas(0),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         values.AbsoluteReplicas(0),
			wantSavedCPU:         0.5,               // 5 replicas × 0.1 cores each
			wantSavedMemory:      320 * 1024 * 1024, // 5 replicas × 64Mi = 320MiB
		},
		{
			name:                 "original replicas set, but not scaled down",
			replicas:             values.AbsoluteReplicas(2),
			originalReplicas:     values.AbsoluteReplicas(5),
			downtimeReplicas:     values.AbsoluteReplicas(0),
			wantOriginalReplicas: values.AbsoluteReplicas(2),
			wantReplicas:         values.AbsoluteReplicas(0),
			wantSavedCPU:         0.2,               // 2 replicas × 0.1 cores
			wantSavedMemory:      128 * 1024 * 1024, // 2 replicas × 64Mi
		},
		{
			name:                 "downscale replicas is not AbsoluteReplicas",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			downtimeReplicas:     values.PercentageReplicas(50),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
			wantSavedCPU:         0.0,
			wantSavedMemory:      0.0,
			wantErr:              &values.InvalidReplicaTypeError{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deploy := &appsv1.Deployment{}
			replicasInt32, _ := test.replicas.AsInt32()
			deploy.Spec.Replicas = new(int32)
			*deploy.Spec.Replicas = replicasInt32

			deploy.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			}

			workload := &replicaScaledWorkload{&deployment{deploy}}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			savedResources, err := workload.ScaleDown(test.downtimeReplicas)

			if test.wantErr != nil {
				var targetErr *values.InvalidReplicaTypeError
				assert.ErrorAs(t, err, &targetErr)

				return
			}

			require.NoError(t, err)

			gotReplicas, err := workload.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, gotReplicas)

			gotOriginal, err := getOriginalReplicas(workload)
			require.NoError(t, err)
			assert.Equal(t, test.wantOriginalReplicas, gotOriginal)

			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)    // CPU tolerance
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5) // Memory tolerance
		})
	}
}
