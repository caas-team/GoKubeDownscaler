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
		replicas             int32
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantReplicas         int32
	}{
		{
			name:                 "scale up",
			replicas:             0,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: nil,
			wantReplicas:         5,
		},
		{
			name:                 "already scaled up",
			replicas:             5,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         5,
		},
		{
			name:                 "orignal replicas not set",
			replicas:             0,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deployment := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
			_ = deployment.setReplicas(test.replicas)

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, deployment)
			}

			err := deployment.ScaleUp()
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
		replicas             int32
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantReplicas         int32
	}{
		{
			name:                 "scale down",
			replicas:             5,
			originalReplicas:     nil,
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         0,
		},
		{
			name:                 "already scaled down",
			replicas:             0,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         0,
		},
		{
			name:                 "orignal replicas set, but not scaled down",
			replicas:             2,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(2),
			wantReplicas:         0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deployment := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
			_ = deployment.setReplicas(test.replicas)

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, deployment)
			}

			err := deployment.ScaleDown(values.AbsoluteReplicas(0))
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

func TestReplicaScaledWorkload_InvalidReplicaTypes(t *testing.T) {
	t.Parallel()

	t.Run("ScaleUp fails if originalReplicas is not AbsoluteReplicas", func(t *testing.T) {
		t.Parallel()

		deployment := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
		_ = deployment.setReplicas(0)

		setOriginalReplicas(values.PercentageReplicas(50), deployment)

		err := deployment.ScaleUp()
		require.Error(t, err)
		require.Contains(t, err.Error(), "percentage")
	})

	t.Run("ScaleDown fails if downscaleReplicas is not AbsoluteReplicas", func(t *testing.T) {
		t.Parallel()

		deployment := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
		_ = deployment.setReplicas(5)

		err := deployment.ScaleDown(values.PercentageReplicas(50))
		require.Error(t, err)
		require.Contains(t, err.Error(), "percentage")
	})
}
