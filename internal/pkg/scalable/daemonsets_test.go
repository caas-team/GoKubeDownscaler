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

func TestDaemonSet_ScaleUp(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			deamonset := daemonSet{&appsv1.DaemonSet{}}

			if test.labelSet {
				deamonset.Spec.Template.Spec.NodeSelector = map[string]string{labelMatchNone: "true"}
			}

			err := deamonset.ScaleUp()
			require.NoError(t, err)

			_, ok := deamonset.Spec.Template.Spec.NodeSelector[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)
		})
	}
}

func TestDaemonSet_ScaleDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		labelSet         bool
		wantLabelSet     bool
		currentScheduled int32
		requestsCPU      string
		requestsMemory   string
		wantSavedCPU     float64
		wantSavedMemory  float64
	}{
		{
			name:             "scale down",
			labelSet:         false,
			wantLabelSet:     true,
			currentScheduled: 3,
			requestsCPU:      "100m",    // 0.1 CPU
			requestsMemory:   "200Mi",   // 200 MiB
			wantSavedCPU:     0.3,       // 0.1 * 3
			wantSavedMemory:  629145600, // 200Mi * 3
		},
		{
			name:             "already scaled down",
			labelSet:         true,
			wantLabelSet:     true,
			currentScheduled: 2,
			requestsCPU:      "50m",     // 0.05 CPU
			requestsMemory:   "100Mi",   // 100 MiB
			wantSavedCPU:     0.1,       // 0.05 * 2
			wantSavedMemory:  209715200, // 100Mi * 2
		},
		{
			name:             "scale down with no resource requests",
			labelSet:         false,
			wantLabelSet:     true,
			currentScheduled: 2,
			requestsCPU:      "",
			requestsMemory:   "",
			wantSavedCPU:     0.0,
			wantSavedMemory:  0.0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			daemonset := daemonSet{&appsv1.DaemonSet{}}
			daemonset.Status.CurrentNumberScheduled = test.currentScheduled

			// set container requests
			if test.requestsCPU != "" || test.requestsMemory != "" {
				reqs := corev1.ResourceList{}
				if test.requestsCPU != "" {
					reqs[corev1.ResourceCPU] = resource.MustParse(test.requestsCPU)
				}

				if test.requestsMemory != "" {
					reqs[corev1.ResourceMemory] = resource.MustParse(test.requestsMemory)
				}

				daemonset.Spec.Template.Spec.Containers = []corev1.Container{
					{Resources: corev1.ResourceRequirements{Requests: reqs}},
				}
			}

			if test.labelSet {
				daemonset.Spec.Template.Spec.NodeSelector = map[string]string{labelMatchNone: "true"}
			}

			totalSavedCPU, totalSavedMemory, err := daemonset.ScaleDown(values.AbsoluteReplicas(0))
			require.NoError(t, err)

			_, ok := daemonset.Spec.Template.Spec.NodeSelector[labelMatchNone]
			assert.Equal(t, test.wantLabelSet, ok)

			assert.InDelta(t, test.wantSavedCPU, totalSavedCPU, 0.0001)
			assert.InDelta(t, test.wantSavedMemory, totalSavedMemory, 1e5)
		})
	}
}
