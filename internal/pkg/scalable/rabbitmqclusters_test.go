//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestRabbitmqCluster builds a rabbitmqCluster with spec.replicas set to the given raw value.
// Pass nil to omit spec.replicas entirely.
func newTestRabbitmqCluster(replicasVal any) *rabbitmqCluster {
	obj := map[string]any{
		"apiVersion": "rabbitmq.com/v1beta1",
		"kind":       "RabbitmqCluster",
		"metadata": map[string]any{
			"name":      "test-rabbitmqcluster",
			"namespace": "default",
		},
		"spec": map[string]any{},
	}

	if replicasVal != nil {
		obj["spec"].(map[string]any)["replicas"] = replicasVal
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(rabbitmqClusterGVK)

	return &rabbitmqCluster{Unstructured: u}
}

func TestRabbitmqCluster_GetReplicas(t *testing.T) {
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
			replicasVal:  int64(1),
			wantReplicas: values.AbsoluteReplicas(1),
		},
		{
			name:         "zero replicas is a legal value for this operator",
			replicasVal:  int64(0),
			wantReplicas: values.AbsoluteReplicas(0),
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

			w := newTestRabbitmqCluster(test.replicasVal)

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

func TestRabbitmqCluster_SetReplicas(t *testing.T) {
	t.Parallel()

	w := newTestRabbitmqCluster(float64(3))

	require.NoError(t, w.setReplicas(0))

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), got)
}

func TestRabbitmqCluster_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	cluster := newTestRabbitmqCluster(float64(3))
	cluster.Object["spec"].(map[string]any)["resources"] = map[string]any{
		"requests": map[string]any{
			"cpu":    "500m",
			"memory": "1Gi",
		},
	}

	saved := cluster.getSavedResourcesRequests(3)

	assert.InDelta(t, 1.5, saved.TotalCPU(), 0.0001)
	assert.InDelta(t, 3*1024*1024*1024, saved.TotalMemory(), 0.0001)
}

func TestRabbitmqCluster_GetSavedResourcesRequests_NoResources(t *testing.T) {
	t.Parallel()

	w := newTestRabbitmqCluster(float64(3))

	saved := w.getSavedResourcesRequests(3)

	assert.InDelta(t, 0, saved.TotalCPU(), 0.0001)
	assert.InDelta(t, 0, saved.TotalMemory(), 0.0001)
}

func TestRabbitmqCluster_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestRabbitmqCluster(float64(5))
	rsw := &replicaScaledWorkload{replicaScaledResource: original}

	copyWorkload, err := rsw.Copy()
	require.NoError(t, err)

	require.NoError(t, original.setReplicas(0))

	origReplicas, err := original.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), origReplicas)

	copyRSW, ok := copyWorkload.(*replicaScaledWorkload)
	require.True(t, ok)
	copyRC, ok := copyRSW.replicaScaledResource.(*rabbitmqCluster)
	require.True(t, ok)

	copyReplicas, err := copyRC.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(5), copyReplicas)
}

func TestRabbitmqCluster_GVK(t *testing.T) {
	t.Parallel()

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(rabbitmqClusterGVK)
	w := &rabbitmqCluster{Unstructured: u}

	assert.Equal(t, rabbitmqClusterGVK.Group, w.GroupVersionKind().Group)
	assert.Equal(t, rabbitmqClusterGVK.Version, w.GroupVersionKind().Version)
	assert.Equal(t, rabbitmqClusterGVK.Kind, w.GroupVersionKind().Kind)
}

func TestIsSupportedOwnerKind_RabbitmqCluster(t *testing.T) {
	t.Parallel()

	// The operator's StatefulSet carries a controller ownerReference to the RabbitmqCluster.
	// It must be filtered out so the CR is the only thing scaled.
	assert.True(t, isSupportedOwnerKind(rabbitmqClusterKind))
}
