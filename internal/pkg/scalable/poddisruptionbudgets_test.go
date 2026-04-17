package scalable

import (
	"errors"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestPodDisruptionBudget_ScaleUp(t *testing.T) {
	t.Parallel()

	replicasUpscaled := intstr.FromInt32(5)
	replicasDownscaled := intstr.FromInt32(0)
	percentileUpscaled := intstr.FromString("50%")
	percentileDownscaled := intstr.FromString("0%")
	tests := []struct {
		name                 string
		minAvailable         *intstr.IntOrString
		maxUnavailable       *intstr.IntOrString
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantMinAvailable     *intstr.IntOrString
		wantMaxUnavailable   *intstr.IntOrString
		wantUpdateNeeded     bool
	}{
		{
			name:                 "minAvailable scale up",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: nil,
			wantMinAvailable:     &replicasUpscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "minAvailable already scaled up",
			minAvailable:         &replicasUpscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &replicasUpscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "minAvailable orignal replicas not set",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "minAvailable percentile",
			minAvailable:         &percentileDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     values.PercentageReplicas(50),
			wantOriginalReplicas: nil,
			wantMinAvailable:     &percentileUpscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "minAvailable percentile already scaled up",
			minAvailable:         &percentileUpscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     &percentileUpscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "maxUnavailable scale up",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasUpscaled,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "maxUnavailable already scaled up",
			minAvailable:         nil,
			maxUnavailable:       &replicasUpscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasUpscaled,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "maxUnavailable orignal replicas not set",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "maxUnavailable percentile",
			minAvailable:         nil,
			maxUnavailable:       &percentileDownscaled,
			originalReplicas:     values.PercentageReplicas(50),
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &percentileUpscaled,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "maxUnavailable percentile already scaled up",
			minAvailable:         nil,
			maxUnavailable:       &percentileUpscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &percentileUpscaled,
			wantUpdateNeeded:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pdb := &podDisruptionBudget{&policy.PodDisruptionBudget{}}
			pdb.Spec.MaxUnavailable = test.maxUnavailable
			pdb.Spec.MinAvailable = test.minAvailable

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, pdb)
			}

			updateNeeded, err := pdb.ScaleUp()
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

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

			assert.Equal(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestPodDisruptionBudget_ScaleDown(t *testing.T) {
	t.Parallel()

	replicasUpscaled := intstr.FromInt32(5)
	replicasUpscaled2 := intstr.FromInt32(2)
	replicasDownscaled := intstr.FromInt32(0)
	percentileUpscaled := intstr.FromString("50%")
	tests := []struct {
		name                 string
		minAvailable         *intstr.IntOrString
		maxUnavailable       *intstr.IntOrString
		originalReplicas     values.Replicas
		wantOriginalReplicas values.Replicas
		wantMinAvailable     *intstr.IntOrString
		wantMaxUnavailable   *intstr.IntOrString
		wantUpdateNeeded     bool
	}{
		{
			name:                 "minAvailable scale down",
			minAvailable:         &replicasUpscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "minAvailable already scaled down",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "minAvailable orignal replicas set, but not scaled down",
			minAvailable:         &replicasUpscaled2,
			maxUnavailable:       nil,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(2),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "minAvailable percentileUpscaled",
			minAvailable:         &percentileUpscaled,
			maxUnavailable:       nil,
			originalReplicas:     nil,
			wantOriginalReplicas: values.PercentageReplicas(50),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "minAvailable percentileUpscaled already scaled down",
			minAvailable:         &replicasDownscaled,
			maxUnavailable:       nil,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantMinAvailable:     &replicasDownscaled,
			wantMaxUnavailable:   nil,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "maxUnavailable scale down",
			minAvailable:         nil,
			maxUnavailable:       &replicasUpscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "maxUnavailable already scaled down",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(5),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
			wantUpdateNeeded:     false,
		},
		{
			name:                 "maxUnavailable orignal replicas set, but not scaled down",
			minAvailable:         nil,
			maxUnavailable:       &replicasUpscaled2,
			originalReplicas:     values.AbsoluteReplicas(5),
			wantOriginalReplicas: values.AbsoluteReplicas(2),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "maxUnavailable percentileUpscaled",
			minAvailable:         nil,
			maxUnavailable:       &percentileUpscaled,
			originalReplicas:     nil,
			wantOriginalReplicas: values.PercentageReplicas(50),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
			wantUpdateNeeded:     true,
		},
		{
			name:                 "maxUnavailable percentileUpscaled already scaled down",
			minAvailable:         nil,
			maxUnavailable:       &replicasDownscaled,
			originalReplicas:     values.AbsoluteReplicas(50),
			wantOriginalReplicas: values.AbsoluteReplicas(50),
			wantMinAvailable:     nil,
			wantMaxUnavailable:   &replicasDownscaled,
			wantUpdateNeeded:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pdb := &podDisruptionBudget{&policy.PodDisruptionBudget{}}
			pdb.Spec.MaxUnavailable = test.maxUnavailable
			pdb.Spec.MinAvailable = test.minAvailable

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, pdb)
			}

			_, updateNeeded, err := pdb.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

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

			assert.Equal(t, test.wantOriginalReplicas.String(), oringalReplicas.String())
		})
	}
}
