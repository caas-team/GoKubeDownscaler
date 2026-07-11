package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNodeSelectorScaledWorkload_ScaleUp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		labelSet         bool
		originalReplicas values.Replicas
		wantLabelSet     bool
		wantUpdateNeeded bool
	}{
		{
			name:             "scale up",
			labelSet:         true,
			originalReplicas: values.BooleanReplicas(false),
			wantLabelSet:     false,
			wantUpdateNeeded: true,
		},
		{
			name:             "already scaled up",
			labelSet:         false,
			originalReplicas: nil,
			wantLabelSet:     false,
			wantUpdateNeeded: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			workload := &nodeSelectorScaledWorkload{nodeSelectorScaledResource: &daemonSet{&appsv1.DaemonSet{}}}

			if test.labelSet {
				workload.setNodeSelector(map[string]string{labelMatchNone: labelMatchNoneValue})
			}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			updateNeeded, err := workload.ScaleUp()
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

			_, ok := workload.getNodeSelector()[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)
		})
	}
}

func TestNodeSelectorScaledWorkload_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		labelSet         bool
		originalReplicas values.Replicas
		currentScheduled int32
		requestsCPU      string
		requestsMemory   string
		wantLabelSet     bool
		wantSavedCPU     float64
		wantSavedMemory  float64
		wantUpdateNeeded bool
	}{
		{
			name:             "scale down",
			labelSet:         false,
			originalReplicas: nil,
			currentScheduled: 3,
			requestsCPU:      "100m",  // 0.1 CPU
			requestsMemory:   "200Mi", // 200 MiB
			wantLabelSet:     true,
			wantSavedCPU:     0.3,       // 0.1 * 3
			wantSavedMemory:  629145600, // 200Mi * 3
			wantUpdateNeeded: true,
		},
		{
			// label is set AND originalReplicas IS set: we already scaled it down in a previous cycle.
			name:             "already scaled down",
			labelSet:         true,
			originalReplicas: values.BooleanReplicas(false),
			currentScheduled: 2,
			requestsCPU:      "50m",   // 0.05 CPU
			requestsMemory:   "100Mi", // 100 MiB
			wantLabelSet:     true,
			wantSavedCPU:     0.1,       // 0.05 * 2
			wantSavedMemory:  209715200, // 100Mi * 2
			wantUpdateNeeded: false,
		},
		{
			// label is set but originalReplicas is NOT set: label was set externally before the downscaler.
			name:             "already at target scale down state",
			labelSet:         true,
			originalReplicas: nil,
			currentScheduled: 2,
			requestsCPU:      "50m",
			requestsMemory:   "100Mi",
			wantLabelSet:     true,
			wantSavedCPU:     0.0,
			wantSavedMemory:  0.0,
			wantUpdateNeeded: false,
		},
		{
			name:             "scale down with no resource requests",
			labelSet:         false,
			originalReplicas: nil,
			currentScheduled: 2,
			requestsCPU:      "",
			requestsMemory:   "",
			wantLabelSet:     true,
			wantSavedCPU:     0.0,
			wantSavedMemory:  0.0,
			wantUpdateNeeded: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			testDaemonSet := &daemonSet{&appsv1.DaemonSet{}}
			testDaemonSet.Status.CurrentNumberScheduled = test.currentScheduled

			if test.requestsCPU != "" || test.requestsMemory != "" {
				reqs := corev1.ResourceList{}
				if test.requestsCPU != "" {
					reqs[corev1.ResourceCPU] = resource.MustParse(test.requestsCPU)
				}

				if test.requestsMemory != "" {
					reqs[corev1.ResourceMemory] = resource.MustParse(test.requestsMemory)
				}

				testDaemonSet.Spec.Template.Spec.Containers = []corev1.Container{{Resources: corev1.ResourceRequirements{Requests: reqs}}}
			}

			workload := &nodeSelectorScaledWorkload{nodeSelectorScaledResource: testDaemonSet}

			if test.labelSet {
				workload.setNodeSelector(map[string]string{labelMatchNone: labelMatchNoneValue})
			}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			savedResources, updateNeeded, err := workload.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

			_, ok := workload.getNodeSelector()[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)

			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5)
		})
	}
}
