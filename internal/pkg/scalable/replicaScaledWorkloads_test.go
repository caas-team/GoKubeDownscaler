package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func TestReplicaScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		replicas             values.Replicas
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantReplicas         values.Replicas
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
			name:                 "orignal replicas not set",
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
			if _, ok := test.originalReplicas.(values.PercentageReplicas); ok {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "percentage")

				return
			}

			require.NoError(t, err)

			replicas, err := deployment.getReplicas()

			if assert.NoError(t, err) {
				assert.Equal(t, test.wantReplicas, replicas)
			}

			oringalReplicas, err := getOriginalReplicas(deployment)

			var originalReplicasUnsetErr *OriginalReplicasUnsetError

			if ok := errors.As(err, &originalReplicasUnsetErr); !ok { // ignore getOriginalReplicas being unset
				require.NoError(t, err) // Scaling set OrignialReplicas to faulty value
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
		wantOriginalReplicas values.Replicas
		wantReplicas         values.Replicas
	}{
		{
			name:                 "scale down",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "already scaled down",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "orignal replicas set, but not scaled down",
			replicas:             values.AbsoluteReplicas(2),
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(2),
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "downscale replicas is not AbsoluteReplicas",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
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

			var downscaleReplica values.Replicas = values.AbsoluteReplicas(0)
			if test.name == "downscale replicas is not AbsoluteReplicas" {
				downscaleReplica = values.PercentageReplicas(50)
			}

			err := deployment.ScaleDown(downscaleReplica)

			if _, ok := downscaleReplica.(values.PercentageReplicas); ok {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to convert downscale replicas")

				return
			}

			require.NoError(t, err)

			replicas, err := deployment.getReplicas()
			if assert.NoError(t, err) {
				assert.Equal(t, test.wantReplicas, replicas)
			}

			oringalReplicas, err := getOriginalReplicas(deployment)
			require.NoError(t, err) // Scaling set OrignialReplicas to faulty or unset value
			assert.Equal(t, test.wantOriginalReplicas.String(), oringalReplicas.String())
		})
	}
}
