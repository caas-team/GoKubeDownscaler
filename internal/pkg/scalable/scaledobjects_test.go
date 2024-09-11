package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestScaledObjects_ScaleUp(t *testing.T) {
	tests := []struct {
		name                 string
		replicas             int
		originalReplicas     *int
		wantOriginalReplicas *int
		wantReplicas         int
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
		{
			name:                 "scale up replicas undefined",
			replicas:             values.Undefined,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: nil,
			wantReplicas:         5,
		},
		{
			name:                 "already scaled up replicas undefined",
			replicas:             values.Undefined,
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         values.Undefined,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			so := &scaledObject{&kedav1alpha1.ScaledObject{}}
			if test.replicas != values.Undefined {
				so.setPauseAnnotation(test.replicas)
			}
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, so)
			}

			err := so.ScaleUp()
			assert.NoError(t, err)
			gotReplicas, err := so.getPauseAnnotation()
			assert.NoError(t, err) // Scaling set PauseAnnotation to faulty value
			assert.Equal(t, test.wantReplicas, gotReplicas)
			oringalReplicas, err := getOriginalReplicas(so)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestScaledObjects_ScaleDown(t *testing.T) {
	tests := []struct {
		name                 string
		replicas             int
		originalReplicas     *int
		wantOriginalReplicas *int
		wantReplicas         int
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
			originalReplicas:     nil,
			wantOriginalReplicas: nil,
			wantReplicas:         0,
		},
		{
			name:                 "orignal replicas set, but not scaled down",
			replicas:             2,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(2),
			wantReplicas:         0,
		},
		{
			name:                 "scale down replicas undefined",
			replicas:             values.Undefined,
			originalReplicas:     nil,
			wantOriginalReplicas: intAsPointer(values.Undefined),
			wantReplicas:         0,
		},
		{
			name:                 "orignal replicas set, but not scaled down replicas undefined",
			replicas:             values.Undefined,
			originalReplicas:     intAsPointer(5),
			wantOriginalReplicas: intAsPointer(values.Undefined),
			wantReplicas:         0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			so := &scaledObject{&kedav1alpha1.ScaledObject{}}
			if test.replicas != values.Undefined {
				so.setPauseAnnotation(test.replicas)
			}
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, so)
			}

			err := so.ScaleDown(0)
			assert.NoError(t, err)
			gotReplicas, err := so.getPauseAnnotation()
			assert.NoError(t, err) // Scaling set PauseAnnotation to faulty value
			assert.Equal(t, test.wantReplicas, gotReplicas)
			oringalReplicas, err := getOriginalReplicas(so)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
