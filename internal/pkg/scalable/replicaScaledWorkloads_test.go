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
		wantErr              error
	}{
		{
			name:                 "scale down",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			downtimeReplicas:     values.AbsoluteReplicas(0),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "already scaled down",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.AbsoluteReplicas(5),
			downtimeReplicas:     values.AbsoluteReplicas(0),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "original replicas set, but not scaled down",
			replicas:             values.AbsoluteReplicas(2),
			originalReplicas:     values.AbsoluteReplicas(5),
			downtimeReplicas:     values.AbsoluteReplicas(0),
			wantOriginalReplicas: values.AbsoluteReplicas(2),
			wantReplicas:         values.AbsoluteReplicas(0),
		},
		{
			name:                 "downscale replicas is not AbsoluteReplicas",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			downtimeReplicas:     values.PercentageReplicas(50),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
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

			err := deployment.ScaleDown(test.downtimeReplicas)
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
			require.NoError(t, err)
			assert.Equal(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
