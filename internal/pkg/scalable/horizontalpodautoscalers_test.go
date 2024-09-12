package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/autoscaling/v2"
)

func TestHPA_ScaleUp(t *testing.T) {
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
			hpa := &horizontalPodAutoscaler{&appsv1.HorizontalPodAutoscaler{}}
			hpa.Spec.MinReplicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, hpa)
			}

			err := hpa.ScaleUp()
			assert.NoError(t, err)
			if assert.NotNil(t, hpa.Spec.MinReplicas) {
				assert.Equal(t, test.wantReplicas, *hpa.Spec.MinReplicas)
			}
			oringalReplicas, err := getOriginalReplicas(hpa)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestHPA_ScaleDown(t *testing.T) {
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
			wantReplicas:         1,
		},
		{
			name:                 "already scaled down",
			replicas:             1,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(5),
			wantReplicas:         1,
		},
		{
			name:                 "orignal replicas set, but not scaled down",
			replicas:             2,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(2),
			wantReplicas:         1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hpa := &horizontalPodAutoscaler{&appsv1.HorizontalPodAutoscaler{}}
			hpa.Spec.MinReplicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, hpa)
			}

			err := hpa.ScaleDown(1)
			assert.NoError(t, err)
			if assert.NotNil(t, hpa.Spec.MinReplicas) {
				assert.Equal(t, test.wantReplicas, *hpa.Spec.MinReplicas)
			}
			oringalReplicas, err := getOriginalReplicas(hpa)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
