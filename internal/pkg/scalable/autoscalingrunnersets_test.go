//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestAutoscalingRunnerSet builds an autoscalingRunnerSet with spec.minRunners set to the given raw value.
// Pass nil to omit spec.minRunners entirely.
func newTestAutoscalingRunnerSet(minRunnersVal any) *autoscalingRunnerSet {
	obj := map[string]any{
		"apiVersion": "actions.github.com/v1alpha1",
		"kind":       "AutoscalingRunnerSet",
		"metadata": map[string]any{
			"name":      "test-runnerset",
			"namespace": "default",
		},
		"spec": map[string]any{
			"listenerMetrics": map[string]any{"gauge": map[string]any{}},
		},
	}

	if minRunnersVal != nil {
		obj["spec"].(map[string]any)["minRunners"] = minRunnersVal
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(autoscalingRunnerSetGVK)

	return &autoscalingRunnerSet{Unstructured: u}
}

func TestAutoscalingRunnerSet_GetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		minRunnersV  any
		wantReplicas values.Replicas
		wantErr      bool
	}{
		{
			name:         "float64 minRunners (API-server JSON)",
			minRunnersV:  float64(3),
			wantReplicas: values.AbsoluteReplicas(3),
		},
		{
			name:         "int64 minRunners",
			minRunnersV:  int64(5),
			wantReplicas: values.AbsoluteReplicas(5),
		},
		{
			name:    "absent spec.minRunners",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestAutoscalingRunnerSet(test.minRunnersV)

			got, err := w.getReplicas()

			if test.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, got)
		})
	}
}

func TestAutoscalingRunnerSet_SetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initial      any
		newReplicas  int32
		wantReplicas values.Replicas
	}{
		{
			name:         "scale down to zero",
			initial:      float64(3),
			newReplicas:  0,
			wantReplicas: values.AbsoluteReplicas(0),
		},
		{
			name:         "scale up",
			initial:      float64(0),
			newReplicas:  4,
			wantReplicas: values.AbsoluteReplicas(4),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestAutoscalingRunnerSet(test.initial)

			require.NoError(t, w.setReplicas(test.newReplicas))

			got, err := w.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, got)
		})
	}
}

// TestAutoscalingRunnerSet_SetReplicas_PreservesUnknownFields guards the reason this workload is unstructured:
// fields unknown to the pinned typed ARC API (such as spec.listenerMetrics) must survive a replica change.
func TestAutoscalingRunnerSet_SetReplicas_PreservesUnknownFields(t *testing.T) {
	t.Parallel()

	w := newTestAutoscalingRunnerSet(float64(3))

	require.NoError(t, w.setReplicas(0))

	listenerMetrics, found, err := unstructured.NestedMap(w.Object, "spec", "listenerMetrics")
	require.NoError(t, err)
	require.True(t, found)
	assert.Contains(t, listenerMetrics, "gauge")
}

func TestAutoscalingRunnerSet_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestAutoscalingRunnerSet(float64(5))
	rsw := &replicaScaledWorkload{replicaScaledResource: original}

	copyWorkload, err := rsw.Copy()
	require.NoError(t, err)

	require.NoError(t, original.setReplicas(0))

	origReplicas, err := original.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), origReplicas)

	copyRSW, ok := copyWorkload.(*replicaScaledWorkload)
	require.True(t, ok)
	copyARS, ok := copyRSW.replicaScaledResource.(*autoscalingRunnerSet)
	require.True(t, ok)

	copyReplicas, err := copyARS.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(5), copyReplicas)
}

func TestAutoscalingRunnerSet_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		containers []any
		diff       int32
		wantCPU    float64
		wantMemory float64
	}{
		{
			name: "single container with requests",
			containers: []any{
				map[string]any{
					"name": "runner",
					"resources": map[string]any{
						"requests": map[string]any{"cpu": "500m", "memory": "1Gi"},
					},
				},
			},
			diff:       2,
			wantCPU:    1.0,
			wantMemory: 2 * 1024 * 1024 * 1024,
		},
		{
			name: "container without requests saves nothing",
			containers: []any{
				map[string]any{"name": "runner"},
			},
			diff:       2,
			wantCPU:    0,
			wantMemory: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestAutoscalingRunnerSet(float64(2))
			require.NoError(t, unstructured.SetNestedSlice(w.Object, test.containers, "spec", "template", "spec", "containers"))

			saved := w.getSavedResourcesRequests(test.diff)

			assert.InDelta(t, test.wantCPU, saved.TotalCPU(), 1e-9)
			assert.InDelta(t, test.wantMemory, saved.TotalMemory(), 1e5)
		})
	}
}

func TestAutoscalingRunnerSet_GetSavedResourcesRequests_MissingResources(t *testing.T) {
	t.Parallel()

	w := newTestAutoscalingRunnerSet(float64(2))

	saved := w.getSavedResourcesRequests(1)

	assert.InDelta(t, 0, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 0, saved.TotalMemory(), 1e-9)
}

func TestAutoscalingRunnerSet_GVK(t *testing.T) {
	t.Parallel()

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(autoscalingRunnerSetGVK)
	w := &autoscalingRunnerSet{Unstructured: u}

	assert.Equal(t, autoscalingRunnerSetGVK.Group, w.GroupVersionKind().Group)
	assert.Equal(t, autoscalingRunnerSetGVK.Version, w.GroupVersionKind().Version)
	assert.Equal(t, autoscalingRunnerSetGVK.Kind, w.GroupVersionKind().Kind)
}
