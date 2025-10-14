package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
		name            string
		suspend         *bool
		parallelism     int32
		cpuRequest      string
		memRequest      string
		wantSuspend     *bool
		wantSavedCPU    float64
		wantSavedMemory float64
	}{
		{
			name:            "scale down",
			suspend:         boolAsPointer(false),
			parallelism:     2,
			cpuRequest:      "250m",
			memRequest:      "128Mi",
			wantSuspend:     boolAsPointer(true),
			wantSavedCPU:    0.25 * 2,              // 250m * 2
			wantSavedMemory: 128 * 1024 * 1024 * 2, // 128Mi * 2
		},
		{
			name:            "already scaled down",
			suspend:         boolAsPointer(true),
			parallelism:     2,
			cpuRequest:      "250m",
			memRequest:      "128Mi",
			wantSuspend:     boolAsPointer(true),
			wantSavedCPU:    0.25 * 2,
			wantSavedMemory: 128 * 1024 * 1024 * 2,
		},
		{
			name:            "suspend unset",
			suspend:         nil,
			parallelism:     2,
			cpuRequest:      "250m",
			memRequest:      "128Mi",
			wantSuspend:     boolAsPointer(true),
			wantSavedCPU:    0.25 * 2,
			wantSavedMemory: 128 * 1024 * 1024 * 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cronjob := cronJob{&batch.CronJob{}}
			cronjob.Spec.Suspend = test.suspend
			cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(test.cpuRequest),
							corev1.ResourceMemory: resource.MustParse(test.memRequest),
						},
					},
				},
			}
			cronjob.Spec.JobTemplate.Spec.Parallelism = &test.parallelism

			s := suspendScaledWorkload{&cronjob}

			savedResources, err := s.ScaleDown(nil)
			require.NoError(t, err)

			assertBoolPointerEqual(t, test.wantSuspend, cronjob.Spec.Suspend)
			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5)
		})
	}
}
