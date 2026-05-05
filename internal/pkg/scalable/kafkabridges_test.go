//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestKafkaBridge builds a kafkaBridge with spec.replicas set to the given raw value.
// Pass nil to omit spec.replicas entirely.
func newTestKafkaBridge(replicasVal any) *kafkaBridge {
	obj := map[string]any{
		"apiVersion": "kafka.strimzi.io/v1",
		"kind":       "KafkaBridge",
		"metadata": map[string]any{
			"name":      "test-kafkabridge",
			"namespace": "default",
		},
		"spec": map[string]any{},
	}

	if replicasVal != nil {
		obj["spec"].(map[string]any)["replicas"] = replicasVal
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(kafkaBridgeGVK)

	return &kafkaBridge{Unstructured: u}
}

func TestKafkaBridge_GetReplicas(t *testing.T) {
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

			w := newTestKafkaBridge(test.replicasVal)

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

func TestKafkaBridge_SetReplicas(t *testing.T) {
	t.Parallel()

	w := newTestKafkaBridge(float64(3))

	require.NoError(t, w.setReplicas(0))

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), got)
}

func TestKafkaBridge_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestKafkaBridge(float64(5))
	rsw := &replicaScaledWorkload{replicaScaledResource: original}

	copyWorkload, err := rsw.Copy()
	require.NoError(t, err)

	require.NoError(t, original.setReplicas(0))

	origReplicas, err := original.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), origReplicas)

	copyRSW, ok := copyWorkload.(*replicaScaledWorkload)
	require.True(t, ok)
	copyKB, ok := copyRSW.replicaScaledResource.(*kafkaBridge)
	require.True(t, ok)

	copyReplicas, err := copyKB.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(5), copyReplicas)
}

func TestKafkaBridge_GVK(t *testing.T) {
	t.Parallel()

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(kafkaBridgeGVK)
	w := &kafkaBridge{Unstructured: u}

	assert.Equal(t, kafkaBridgeGVK.Group, w.GroupVersionKind().Group)
	assert.Equal(t, kafkaBridgeGVK.Version, w.GroupVersionKind().Version)
	assert.Equal(t, kafkaBridgeGVK.Kind, w.GroupVersionKind().Kind)
}
