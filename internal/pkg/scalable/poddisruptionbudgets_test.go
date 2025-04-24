package scalable

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPodDisruptionBudget_ScaleUp(t *testing.T) {
	t.Parallel()

	replicasUpscaled := intstr.FromInt32(5)
	replicasDownscaled := intstr.FromInt32(0)
	percentile := intstr.FromString("50%")
	tests := []struct {
		name                 string
		minAvailable         *intstr.IntOrString
		maxUnavailable       *intstr.IntOrString
		originalReplicas     *int32
		wantOriginalReplicas *int32
		wantMinAvailable     *intstr.IntOrString
		wantMaxUnavailable   *intstr.IntOrString
	}{
		{
			name:                 "minAvailable scale up",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: nil,
			wantMinAvailable:     &replicasUpscaled,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable already scaled up",
			minAvailable:         &replicasUpscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &replicasUpscaled,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable orignal replicas not set",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable percentile",
			minAvailable:         &percentile,
			maxUnavailable:       nil,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     &percentile,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable percentile already scaled up",
			minAvailable:         &percentile,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &percentile,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "maxUnavailable scale up",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasUpscaled,
		},
		{
			name:                 "maxUnavailable already scaled up",
			minAvailable:         nil,
			maxUnavailable:       &replicasUpscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasUpscaled,
		},
		{
			name:                 "maxUnavailable orignal replicas not set",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
		},
		{
			name:                 "maxUnavailable percentile",
			minAvailable:         nil,
			maxUnavailable:       &percentile,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &percentile,
		},
		{
			name:                 "maxUnavailable percentile already scaled up",
			minAvailable:         nil,
			maxUnavailable:       &percentile,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &percentile,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pdb := &podDisruptionBudget{&policy.PodDisruptionBudget{}}
			pdb.Spec.MaxUnavailable = test.maxUnavailable
			pdb.Spec.MinAvailable = test.minAvailable

			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, pdb)
			}

			err := pdb.ScaleUp()
			require.NoError(t, err)

			if test.wantMaxUnavailable != nil {
				if assert.NotNil(t, pdb.Spec.MaxUnavailable) {
					assert.Equal(t, *test.wantMaxUnavailable, *pdb.Spec.MaxUnavailable)
				}
			}

			if test.wantMinAvailable != nil {
				if assert.NotNil(t, pdb.Spec.MinAvailable) {
					assert.Equal(t, *test.wantMinAvailable, *pdb.Spec.MinAvailable)
				}
			}

			oringalReplicas, err := getOriginalReplicas(pdb)
			var originalReplicasUnsetErr *OriginalReplicasUnsetError

			if ok := errors.As(err, &originalReplicasUnsetErr); !ok { // ignore getOriginalReplicas being unset
				require.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			}

			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestPodDisruptionBudget_ScaleDown(t *testing.T) {
	t.Parallel()

	replicasUpscaled := intstr.FromInt32(5)
	replicasUpscaled2 := intstr.FromInt32(2)
	replicasDownscaled := intstr.FromInt32(0)
	percentile := intstr.FromString("50%")
	tests := []struct {
		name                 string
		minAvailable         *intstr.IntOrString
		maxUnavailable       *intstr.IntOrString
		originalReplicas     *int32
		wantOriginalReplicas *int32
		wantMinAvailable     *intstr.IntOrString
		wantMaxUnavailable   *intstr.IntOrString
	}{
		{
			name:                 "minAvailable scale down",
			minAvailable:         &replicasUpscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable already scaled down",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable orignal replicas set, but not scaled down",
			minAvailable:         &replicasUpscaled2,
			maxUnavailable:       nil,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(2),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable percentile",
			minAvailable:         &percentile,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &percentile,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "minAvailable percentile already scaled down",
			minAvailable:         &percentile,
			maxUnavailable:       nil,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     &percentile,
			wantMaxUnavailable:   nil,
		},
		{
			name:                 "maxUnavailable scale down",
			minAvailable:         nil,
			maxUnavailable:       &replicasUpscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
		},
		{
			name:                 "maxUnavailable already scaled down",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
		},
		{
			name:                 "maxUnavailable orignal replicas set, but not scaled down",
			minAvailable:         nil,
			maxUnavailable:       &replicasUpscaled2,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(2),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
		},
		{
			name:                 "maxUnavailable percentile",
			minAvailable:         nil,
			maxUnavailable:       &percentile,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &percentile,
		},
		{
			name:                 "maxUnavailable percentile already scaled down",
			minAvailable:         nil,
			maxUnavailable:       &percentile,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &percentile,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pdb := &podDisruptionBudget{&policy.PodDisruptionBudget{}}
			pdb.Spec.MaxUnavailable = test.maxUnavailable
			pdb.Spec.MinAvailable = test.minAvailable

			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, pdb)
			}

			err := pdb.ScaleDown(0)
			require.NoError(t, err)

			if test.wantMaxUnavailable != nil {
				if assert.NotNil(t, pdb.Spec.MaxUnavailable) {
					assert.Equal(t, *test.wantMaxUnavailable, *pdb.Spec.MaxUnavailable)
				}
			}

			if test.wantMinAvailable != nil {
				if assert.NotNil(t, pdb.Spec.MinAvailable) {
					assert.Equal(t, *test.wantMinAvailable, *pdb.Spec.MinAvailable)
				}
			}

			oringalReplicas, err := getOriginalReplicas(pdb)
			var originalReplicasUnsetErr *OriginalReplicasUnsetError

			if ok := errors.As(err, &originalReplicasUnsetErr); !ok { // ignore getOriginalReplicas being unset
				require.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			}

			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
