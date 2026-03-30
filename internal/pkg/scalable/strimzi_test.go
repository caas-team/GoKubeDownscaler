//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// newTestStrimziWorkload builds a strimziWorkload with spec.replicas set to the given raw value.
// Pass nil to omit spec.replicas entirely.
func newTestStrimziWorkload(replicasVal any) *strimziWorkload {
	obj := map[string]any{
		"apiVersion": "kafka.strimzi.io/v1beta2",
		"kind":       "KafkaConnect",
		"metadata": map[string]any{
			"name":      "test-kafkaconnect",
			"namespace": "default",
		},
		"spec": map[string]any{},
	}

	if replicasVal != nil {
		obj["spec"].(map[string]any)["replicas"] = replicasVal
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(kafkaConnectGVK)

	return &strimziWorkload{Unstructured: u, gvk: kafkaConnectGVK}
}

func TestStrimziWorkload_GetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		replicasVal  any
		wantReplicas values.Replicas
		wantErr      bool
	}{
		{
			name:         "float64 replicas (API-server JSON)",
			replicasVal:  float64(3),
			wantReplicas: values.AbsoluteReplicas(3),
		},
		{
			name:         "int64 replicas",
			replicasVal:  int64(5),
			wantReplicas: values.AbsoluteReplicas(5),
		},
		{
			name:        "absent spec.replicas",
			replicasVal: nil,
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestStrimziWorkload(test.replicasVal)

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

func TestStrimziWorkload_SetReplicas(t *testing.T) {
	t.Parallel()

	w := newTestStrimziWorkload(float64(3))

	require.NoError(t, w.setReplicas(0))

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), got)
}

func TestStrimziWorkload_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	w := newTestStrimziWorkload(float64(5))
	rsw := &replicaScaledWorkload{replicaScaledResource: w}

	copyWorkload, err := rsw.Copy()
	require.NoError(t, err)

	// Mutate the copy's replicas and verify the original is unchanged.
	require.NoError(t, w.setReplicas(0))

	origReplicas, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), origReplicas)

	copyRSW, ok := copyWorkload.(*replicaScaledWorkload)
	require.True(t, ok)
	copyStrimzi, ok := copyRSW.replicaScaledResource.(*strimziWorkload)
	require.True(t, ok)

	copyReplicas, err := copyStrimzi.getReplicas()
	require.NoError(t, err)
	// Copy should still have original value (5), not the mutated value (0).
	assert.Equal(t, values.AbsoluteReplicas(5), copyReplicas)
}

func TestStrimziWorkload_GVK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		gvk  schema.GroupVersionKind
	}{
		{
			name: "KafkaConnect",
			gvk:  kafkaConnectGVK,
		},
		{
			name: "KafkaMirrorMaker2",
			gvk:  kafkaMirrorMaker2GVK,
		},
		{
			name: "KafkaBridge",
			gvk:  kafkaBridgeGVK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(test.gvk)
			w := &strimziWorkload{Unstructured: u, gvk: test.gvk}

			assert.Equal(t, test.gvk.Group, w.GroupVersionKind().Group)
			assert.Equal(t, test.gvk.Version, w.GroupVersionKind().Version)
			assert.Equal(t, test.gvk.Kind, w.GroupVersionKind().Kind)
		})
	}
}
