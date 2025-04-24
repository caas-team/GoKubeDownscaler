package scalable

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func TestReplicaScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		replicas             int32
		originalReplicas     *int32
		wantOriginalReplicas *int32
		wantReplicas         int32
	}{
		{
			name:                 "scale up",
			replicas:             0,
			originalReplicas:     intAsPointer(5),
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
				setOriginalReplicas(*test.originalReplicas, deployment)
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

			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestReplicaScaledWorkload_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		replicas             int32
		originalReplicas     *int32
		wantOriginalReplicas *int32
		wantReplicas         int32
	}{
		{
			name:                 "scale down",
			replicas:             5,
			originalReplicas:     nil,
			wantOriginalReplicas: intAsPointer(5),
			wantReplicas:         0,
		},
		{
			name:                 "already scaled down",
			replicas:             0,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantReplicas:         0,
		},
		{
			name:                 "orignal replicas set, but not scaled down",
			replicas:             2,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(2),
			wantReplicas:         0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			deployment := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
			_ = deployment.setReplicas(test.replicas)

			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, deployment)
			}

			err := deployment.ScaleDown(0)
			require.NoError(t, err)

			replicas, err := deployment.getReplicas()
			if assert.NoError(t, err) {
				assert.Equal(t, test.wantReplicas, replicas)
			}

			oringalReplicas, err := getOriginalReplicas(deployment)
			require.NoError(t, err) // Scaling set OrignialReplicas to faulty or unset value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
