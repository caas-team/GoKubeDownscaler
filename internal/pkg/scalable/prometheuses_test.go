package scalable

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
)

func TestPrometheus_ScaleUp(t *testing.T) {
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
			p := &prometheus{&monitoringv1.Prometheus{}}
			p.Spec.Replicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, p)
			}

			err := p.ScaleUp()
			assert.NoError(t, err)
			if assert.NotNil(t, p.Spec.Replicas) {
				assert.Equal(t, test.wantReplicas, *p.Spec.Replicas)
			}
			oringalReplicas, err := getOriginalReplicas(p)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestPrometheus_ScaleDown(t *testing.T) {
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
			p := &prometheus{&monitoringv1.Prometheus{}}
			p.Spec.Replicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, p)
			}

			err := p.ScaleDown(0)
			assert.NoError(t, err)
			if assert.NotNil(t, p.Spec.Replicas) {
				assert.Equal(t, test.wantReplicas, *p.Spec.Replicas)
			}
			oringalReplicas, err := getOriginalReplicas(p)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
