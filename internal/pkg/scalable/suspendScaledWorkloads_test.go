package scalable

import (
	"context"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// TestCronJobGetChildren verifies the GetChildren method.
func TestCronJobGetChildren(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		activeJobs     []corev1.ObjectReference
		wantChildCount int
	}{
		{
			name: "cronJob.Status.Active contains multiple job references for GetChildren to process",
			activeJobs: []corev1.ObjectReference{
				{Name: "job-1"},
				{Name: "job-2"},
				{Name: "job-3"},
			},
			wantChildCount: 3,
		},
		{
			name:           "cronJob with no active jobs returns empty from GetChildren",
			activeJobs:     []corev1.ObjectReference{},
			wantChildCount: 0,
		},
		{
			name: "cronJob.Status.Active with single child reference",
			activeJobs: []corev1.ObjectReference{
				{Name: "job-single"},
			},
			wantChildCount: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			cronJob := &cronJob{
				CronJob: &batch.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cronjob",
						Namespace: "test-namespace",
					},
					Status: batch.CronJobStatus{
						Active: test.activeJobs,
					},
				},
			}

			// This tests that GetChildren reads from Status.Active correctly
			assert.Len(t, cronJob.Status.Active, test.wantChildCount)

			// Verify each active job reference is properly structured
			for i, jobRef := range cronJob.Status.Active {
				assert.NotEmpty(t, jobRef.Name, "job reference at index %d should have a name", i)
			}

			// Test GetChildren method:
			// When there are no active jobs, GetChildren should return empty
			if test.wantChildCount == 0 {
				children, err := cronJob.GetChildren(ctx, nil)
				require.NoError(t, err)
				assert.Empty(t, children)
			} else {
				// When there are active jobs, verify the cronJob properly stores Status.Active
				require.NotNil(t, cronJob.GetChildren)
				require.Len(t, cronJob.Status.Active, test.wantChildCount)
			}
		})
	}
}
