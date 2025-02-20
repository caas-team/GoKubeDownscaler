package scalable

import (
	"testing"

	"github.com/stretchr/testify/require"
	batch "k8s.io/api/batch/v1"
)

func TestSuspendScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			cronjob := cronJob{&batch.CronJob{}}
			cronjob.Spec.Suspend = test.suspend
			s := suspendScaledWorkload{&cronjob}

			err := s.ScaleUp()
			require.NoError(t, err)
			assertBoolPointerEqual(t, test.wantSuspend, cronjob.Spec.Suspend)
		})
	}
}

func TestSuspendScaledWorkload_ScaleDown(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			cronjob := cronJob{&batch.CronJob{}}
			cronjob.Spec.Suspend = test.suspend
			s := suspendScaledWorkload{&cronjob}

			err := s.ScaleDown(0)
			require.NoError(t, err)
			assertBoolPointerEqual(t, test.wantSuspend, cronjob.Spec.Suspend)
		})
	}
}
