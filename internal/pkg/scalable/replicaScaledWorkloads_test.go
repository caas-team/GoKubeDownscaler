package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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
		wantUpdateNeeded     bool
		wantErr              error
	}{
		{
			name:                 "scale up",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
			wantUpdateNeeded:     true,
		},
		{
			name:                 "already scaled up",
			replicas:             values.AbsoluteReplicas(5),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(5),
			wantUpdateNeeded:     false,
		},
		{
			name:                 "original replicas not set",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(0),
			wantUpdateNeeded:     false,
		},
		{
			name:                 "original replicas is not AbsoluteReplicas",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     values.PercentageReplicas(50),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(0),
			wantUpdateNeeded:     false,
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

			updateNeeded, err := deployment.ScaleUp()
			var invalidReplicaTypeError *values.InvalidReplicaTypeError

			if errors.As(test.wantErr, &invalidReplicaTypeError) {
				assert.ErrorAs(t, err, &invalidReplicaTypeError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

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
		wantUpdateNeeded     bool
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
			wantUpdateNeeded:     true,
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
			wantUpdateNeeded:     false,
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
			wantUpdateNeeded:     true,
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
			wantUpdateNeeded:     false,
			wantErr:              &values.InvalidReplicaTypeError{},
		},
		{
			name:                 "current replicas below downtime replicas",
			replicas:             values.AbsoluteReplicas(0),
			originalReplicas:     nil,
			downtimeReplicas:     values.AbsoluteReplicas(1),
			wantOriginalReplicas: nil,
			wantReplicas:         values.AbsoluteReplicas(0),
			wantSavedCPU:         0.0,
			wantSavedMemory:      0.0,
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

			savedResources, updateNeeded, err := workload.ScaleDown(test.downtimeReplicas)

			if test.wantErr != nil {
				var targetErr *values.InvalidReplicaTypeError
				assert.ErrorAs(t, err, &targetErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

			gotReplicas, err := workload.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, gotReplicas)

			gotOriginal, err := getOriginalReplicas(workload)
			var unsetErr *OriginalReplicasUnsetError

			if !errors.As(err, &unsetErr) {
				require.NoError(t, err)
			}

			assert.Equal(t, test.wantOriginalReplicas, gotOriginal)

			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)    // CPU tolerance
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5) // Memory tolerance
		})
	}
}

// TestReplicaScaledWorkload_ScaleDown_ScaledObjectUndefinedReplicas ensures that a ScaledObject without a
// paused-replicas annotation (getReplicas reports the util.Undefined sentinel) is still scaled down, instead
// of being skipped as if it were already at or below the downtime target. It also covers the scale-up
// round-trip: the undefined sentinel must be recorded as the original replicas and restored cleanly.
func TestReplicaScaledWorkload_ScaleDown_ScaledObjectUndefinedReplicas(t *testing.T) {
	t.Parallel()

	workload := &replicaScaledWorkload{&scaledObject{&kedav1alpha1.ScaledObject{}}}

	// no paused-replicas annotation means the current replicas are undefined
	current, err := workload.getReplicas()
	require.NoError(t, err)
	currentInt32, err := current.AsInt32()
	require.NoError(t, err)
	require.Equal(t, int32(util.Undefined), currentInt32)

	// scale down: the workload must be paused and the undefined sentinel recorded as the original replicas
	_, updateNeeded, err := workload.ScaleDown(values.AbsoluteReplicas(1))
	require.NoError(t, err)
	assert.True(t, updateNeeded, "scaled object with undefined replicas should be scaled down")

	gotReplicas, err := workload.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(1), gotReplicas)

	gotOriginal, err := getOriginalReplicas(workload)
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(util.Undefined), gotOriginal,
		"the undefined sentinel must be recorded as the original replicas")

	// scale up: the original (undefined) replicas must be restored, removing the paused-replicas annotation
	updateNeeded, err = workload.ScaleUp()
	require.NoError(t, err)
	assert.True(t, updateNeeded, "scaled object should be scaled back up")

	restored, err := workload.getReplicas()
	require.NoError(t, err)
	restoredInt32, err := restored.AsInt32()
	require.NoError(t, err)
	assert.Equal(t, int32(util.Undefined), restoredInt32,
		"scale up must restore the undefined state (no paused-replicas annotation)")

	_, hasOriginal := getOriginalReplicas(workload)
	var unsetErr *OriginalReplicasUnsetError
	assert.ErrorAs(t, hasOriginal, &unsetErr, "original-replicas annotation must be removed after scale up")
}
