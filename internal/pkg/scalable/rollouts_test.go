package scalable

import (
	"testing"

	argov1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestRollout_ScaleUp(t *testing.T) {
	tests := []struct {
		name                 string
		replicas             int32
		originalReplicas     *int
		wantOriginalReplicas *int
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
			r := &rollout{&argov1alpha1.Rollout{}}
			r.Spec.Replicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, r)
			}

			err := r.ScaleUp()
			assert.NoError(t, err)
			if assert.NotNil(t, r.Spec.Replicas) {
				assert.Equal(t, test.wantReplicas, *r.Spec.Replicas)
			}
			oringalReplicas, err := getOriginalReplicas(r)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestRollout_ScaleDown(t *testing.T) {
	tests := []struct {
		name                 string
		replicas             int32
		originalReplicas     *int
		wantOriginalReplicas *int
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
			r := &rollout{&argov1alpha1.Rollout{}}
			r.Spec.Replicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, r)
			}

			err := r.ScaleDown(0)
			assert.NoError(t, err)
			if assert.NotNil(t, r.Spec.Replicas) {
				assert.Equal(t, test.wantReplicas, *r.Spec.Replicas)
			}
			oringalReplicas, err := getOriginalReplicas(r)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
