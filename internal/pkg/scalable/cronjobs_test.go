package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batch "k8s.io/api/batch/v1"
)

func TestCronJob_ScaleUp(t *testing.T) {
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
			cj := cronJob{&batch.CronJob{}}
			cj.Spec.Suspend = test.suspend

			err := cj.ScaleUp()
			assert.NoError(t, err)
			assertBoolPointerEqual(t, test.wantSuspend, cj.Spec.Suspend)
		})
	}
}

func TestCronJob_ScaleDown(t *testing.T) {
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
			cj := cronJob{&batch.CronJob{}}
			cj.Spec.Suspend = test.suspend

			err := cj.ScaleDown(0)
			assert.NoError(t, err)
			assertBoolPointerEqual(t, test.wantSuspend, cj.Spec.Suspend)
		})
	}
}
