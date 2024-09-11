package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestStatefulSets_ScaleUp(t *testing.T) {
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
			ss := &statefulSet{&appsv1.StatefulSet{}}
			ss.Spec.Replicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, ss)
			}

			err := ss.ScaleUp()
			assert.NoError(t, err)
			if assert.NotNil(t, ss.Spec.Replicas) {
				assert.Equal(t, test.wantReplicas, *ss.Spec.Replicas)
			}
			oringalReplicas, err := getOriginalReplicas(ss)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}

func TestStatefulSets_ScaleDown(t *testing.T) {
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
			ss := &statefulSet{&appsv1.StatefulSet{}}
			ss.Spec.Replicas = &test.replicas
			if test.originalReplicas != nil {
				setOriginalReplicas(*test.originalReplicas, ss)
			}

			err := ss.ScaleDown(0)
			assert.NoError(t, err)
			if assert.NotNil(t, ss.Spec.Replicas) {
				assert.Equal(t, test.wantReplicas, *ss.Spec.Replicas)
			}
			oringalReplicas, err := getOriginalReplicas(ss)
			assert.NoError(t, err) // Scaling set OrignialReplicas to faulty value
			assertIntPointerEqual(t, test.wantOriginalReplicas, oringalReplicas)
		})
	}
}
