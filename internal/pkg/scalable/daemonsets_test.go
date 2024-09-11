package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestDaemonSet_ScaleUp(t *testing.T) {
	tests := []struct {
		name         string
		labelSet     bool
		wantLabelSet bool
	}{
		{
			name:         "scale up",
			labelSet:     true,
			wantLabelSet: false,
		},
		{
			name:         "already scaled up",
			labelSet:     false,
			wantLabelSet: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ds := daemonSet{&appsv1.DaemonSet{}}
			if test.labelSet {
				ds.Spec.Template.Spec.NodeSelector = map[string]string{labelMatchNone: "true"}
			}

			err := ds.ScaleUp()
			assert.NoError(t, err)
			_, ok := ds.Spec.Template.Spec.NodeSelector[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)
		})
	}
}

func TestDaemonSet_ScaleDown(t *testing.T) {
	tests := []struct {
		name         string
		labelSet     bool
		wantLabelSet bool
	}{
		{
			name:         "scale down",
			labelSet:     false,
			wantLabelSet: true,
		},
		{
			name:         "already scaled down",
			labelSet:     true,
			wantLabelSet: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ds := daemonSet{&appsv1.DaemonSet{}}
			if test.labelSet {
				ds.Spec.Template.Spec.NodeSelector = map[string]string{labelMatchNone: "true"}
			}

			err := ds.ScaleDown(0)
			assert.NoError(t, err)
			_, ok := ds.Spec.Template.Spec.NodeSelector[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)
		})
	}
}
