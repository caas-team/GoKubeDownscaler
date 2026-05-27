package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestSuspendScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		suspend          *bool
		originalReplicas values.Replicas
		wantSuspend      *bool
		wantUpdateNeeded bool
	}{
		{
			name:             "scale up",
			suspend:          boolAsPointer(true),
			originalReplicas: values.BooleanReplicas(false),
			wantSuspend:      boolAsPointer(false),
			wantUpdateNeeded: true,
		},
		{
			name:             "already scaled up",
			suspend:          boolAsPointer(false),
			originalReplicas: nil,
			wantSuspend:      boolAsPointer(false),
			wantUpdateNeeded: false,
		},
		{
			name:             "suspend unset",
			suspend:          nil,
			originalReplicas: nil,
			wantSuspend:      nil,
			wantUpdateNeeded: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cronjob := cronJob{&batch.CronJob{}}
			cronjob.Spec.Suspend = test.suspend
			suspendedWorkload := suspendScaledWorkload{&cronjob}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, &suspendedWorkload)
			}

			updateNeeded, err := suspendedWorkload.ScaleUp()
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)
			assertBoolPointerEqual(t, test.wantSuspend, cronjob.Spec.Suspend)
		})
	}
}

func TestSuspendScaledWorkload_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		suspend          *bool
		originalReplicas values.Replicas
		parallelism      *int32
		cpuRequest       string
		memRequest       string
		wantSuspend      *bool
		wantSavedCPU     float64
		wantSavedMemory  float64
		wantUpdateNeeded bool
	}{
		{
			name:             "scale down",
			suspend:          boolAsPointer(false),
			originalReplicas: nil,
			parallelism:      int32Ptr(2),
			cpuRequest:       "250m",
			memRequest:       "128Mi",
			wantSuspend:      boolAsPointer(true),
			wantSavedCPU:     0.25 * 2,              // 250m * 2
			wantSavedMemory:  128 * 1024 * 1024 * 2, // 128Mi * 2
			wantUpdateNeeded: true,
		},
		{
			name:             "scale down nil parallelism",
			suspend:          boolAsPointer(false),
			originalReplicas: nil,
			parallelism:      nil,
			cpuRequest:       "250m",
			memRequest:       "128Mi",
			wantSuspend:      boolAsPointer(true),
			wantSavedCPU:     0.25 * 1,              // default parallelism = 1
			wantSavedMemory:  128 * 1024 * 1024 * 1, // default parallelism = 1
			wantUpdateNeeded: true,
		},
		{
			// currentState == targetScaleDownState but originalReplicas is NOT set:
			// workload was already suspended before the downscaler touched it.
			name:             "already at target scale down state",
			suspend:          boolAsPointer(true),
			originalReplicas: nil,
			parallelism:      int32Ptr(2),
			cpuRequest:       "250m",
			memRequest:       "128Mi",
			wantSuspend:      boolAsPointer(true),
			wantSavedCPU:     0,
			wantSavedMemory:  0,
			wantUpdateNeeded: false,
		},
		{
			// currentState == targetScaleDownState AND originalReplicas IS set:
			// workload was already scaled down by the downscaler in a previous cycle.
			name:             "already scaled down",
			suspend:          boolAsPointer(true),
			originalReplicas: values.BooleanReplicas(false),
			parallelism:      int32Ptr(2),
			cpuRequest:       "250m",
			memRequest:       "128Mi",
			wantSuspend:      boolAsPointer(true),
			wantSavedCPU:     0.25 * 2,
			wantSavedMemory:  128 * 1024 * 1024 * 2,
			wantUpdateNeeded: false,
		},
		{
			name:             "suspend unset",
			suspend:          nil,
			originalReplicas: nil,
			parallelism:      int32Ptr(2),
			cpuRequest:       "250m",
			memRequest:       "128Mi",
			wantSuspend:      boolAsPointer(true),
			wantSavedCPU:     0.25 * 2,
			wantSavedMemory:  128 * 1024 * 1024 * 2,
			wantUpdateNeeded: true,
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
			cronjob.Spec.JobTemplate.Spec.Parallelism = test.parallelism

			suspendedWorkload := suspendScaledWorkload{&cronjob}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, &suspendedWorkload)
			}

			savedResources, updateNeeded, err := suspendedWorkload.ScaleDown(nil)
			require.NoError(t, err)

			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)
			assertBoolPointerEqual(t, test.wantSuspend, cronjob.Spec.Suspend)
			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5)
		})
	}
}
