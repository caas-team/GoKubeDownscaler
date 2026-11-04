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

			daemonset := &nodeSelectorScaledWorkload{nodeSelectorScaledResource: &daemonSet{&appsv1.DaemonSet{}}}

			if test.labelSet {
				daemonset.setNodeSelector(map[string]string{labelMatchNone: labelMatchNoneValue})
			}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, daemonset)
			}

			updateNeeded, err := daemonset.scaleUp(nil)
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

			_, ok := daemonset.getNodeSelector()[labelMatchNone]
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

			daemonset := &daemonSet{&appsv1.DaemonSet{}}
			daemonset.Status.CurrentNumberScheduled = test.currentScheduled

			if test.requestsCPU != "" || test.requestsMemory != "" {
				reqs := corev1.ResourceList{}
				if test.requestsCPU != "" {
					reqs[corev1.ResourceCPU] = resource.MustParse(test.requestsCPU)
				}

				if test.requestsMemory != "" {
					reqs[corev1.ResourceMemory] = resource.MustParse(test.requestsMemory)
				}

				daemonset.Spec.Template.Spec.Containers = []corev1.Container{{Resources: corev1.ResourceRequirements{Requests: reqs}}}
			}

			workload := &nodeSelectorScaledWorkload{nodeSelectorScaledResource: daemonset}

			if test.labelSet {
				workload.setNodeSelector(map[string]string{labelMatchNone: labelMatchNoneValue})
			}

			if test.originalReplicas != nil {
				setOriginalReplicas(test.originalReplicas, workload)
			}

			savedResources, updateNeeded, err := workload.scaleDown(values.AbsoluteReplicas(0), nil)
			require.NoError(t, err)
			assert.Equal(t, test.wantUpdateNeeded, updateNeeded)

			_, ok := workload.getNodeSelector()[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)

			assert.InDelta(t, test.wantSavedCPU, savedResources.TotalCPU(), 0.0001)
			assert.InDelta(t, test.wantSavedMemory, savedResources.TotalMemory(), 1e5)
		})
	}
}

func TestNodeSelectorScaledWorkload_ScalingGeneratesPatchData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		labelSet      bool
		current       int32
		requests      map[string]string
		wantDownPatch any
		wantUpPatch   any
	}{
		{
			name:     "scale down adds label",
			labelSet: false,
			current:  3,
			requests: map[string]string{"cpu": "100m", "memory": "200Mi"},
			wantDownPatch: []any{
				map[string]any{"op": "add", "path": "/metadata/annotations", "value": map[string]any{"downscaler/original-replicas": "false"}},
				map[string]any{"op": "add", "path": "/spec/template/spec/nodeSelector", "value": map[string]any{"downscaler/match-none": "true"}},
			},
			wantUpPatch: []any{
				map[string]any{"op": "remove", "path": "/metadata/annotations"},
				map[string]any{"op": "remove", "path": "/spec/template/spec/nodeSelector"},
			},
		},
		{
			name:          "already downscaled keeps label",
			labelSet:      true,
			current:       2,
			requests:      map[string]string{"cpu": "50m", "memory": "100Mi"},
			wantDownPatch: nil,
			wantUpPatch:   nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			daemonset := &daemonSet{&appsv1.DaemonSet{}}

			daemonset.Status.CurrentNumberScheduled = test.current
			if test.requests != nil {
				reqs := corev1.ResourceList{}
				if cpu, ok := test.requests["cpu"]; ok && cpu != "" {
					reqs[corev1.ResourceCPU] = resource.MustParse(cpu)
				}

				if memory, ok := test.requests["memory"]; ok && memory != "" {
					reqs[corev1.ResourceMemory] = resource.MustParse(memory)
				}

				daemonset.Spec.Template.Spec.Containers = []corev1.Container{{Resources: corev1.ResourceRequirements{Requests: reqs}}}
			}

			workload := &nodeSelectorScaledWorkload{nodeSelectorScaledResource: daemonset}
			if test.labelSet {
				workload.setNodeSelector(map[string]string{labelMatchNone: labelMatchNoneValue})
			}

			beforeDownscale := daemonset.DeepCopy()
			summary, err := workload.ScaleDown(values.AbsoluteReplicas(0), nil)
			require.NoError(t, err)
			t.Logf("downscale patchData: %s", summary.PatchData)
			assertJSONPatchTransforms(t, beforeDownscale, summary.PatchData, daemonset.DaemonSet)

			beforeUpscale := daemonset.DeepCopy()
			upscaleSummary, err := workload.ScaleUp(nil)
			require.NoError(t, err)
			t.Logf("upscale patchData: %s", upscaleSummary.PatchData)
			assertJSONPatchTransforms(t, beforeUpscale, upscaleSummary.PatchData, daemonset.DaemonSet)
		})
	}
}
