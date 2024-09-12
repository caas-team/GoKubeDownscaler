package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batch "k8s.io/api/batch/v1"
)

func TestJob_ScaleUp(t *testing.T) {
	tests := []struct {
		name        string
		suspend     *bool
		wantSuspend *bool
	}{
		{
			name:        "scale up",
			suspend:     boolAsPointer(true),
			wantSuspend: boolAsPointer(false),
		},
		{
			name:        "already scaled up",
			suspend:     boolAsPointer(false),
			wantSuspend: boolAsPointer(false),
		},
		{
			name:        "suspend unset",
			suspend:     nil,
			wantSuspend: boolAsPointer(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			j := job{&batch.Job{}}
			j.Spec.Suspend = test.suspend

			err := j.ScaleUp()
			assert.NoError(t, err)
			assertBoolPointerEqual(t, test.wantSuspend, j.Spec.Suspend)
		})
	}
}

func TestJob_ScaleDown(t *testing.T) {
	tests := []struct {
		name        string
		suspend     *bool
		wantSuspend *bool
	}{
		{
			name:        "scale down",
			suspend:     boolAsPointer(false),
			wantSuspend: boolAsPointer(true),
		},
		{
			name:        "already scaled down",
			suspend:     boolAsPointer(true),
			wantSuspend: boolAsPointer(true),
		},
		{
			name:        "suspend unset",
			suspend:     nil,
			wantSuspend: boolAsPointer(true),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			j := job{&batch.Job{}}
			j.Spec.Suspend = test.suspend

			err := j.ScaleDown(0)
			assert.NoError(t, err)
			assertBoolPointerEqual(t, test.wantSuspend, j.Spec.Suspend)
		})
	}
}
