//nolint:dupl // suspend-shaped scaler tests share a near-identical table shape
package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestCnpgCluster builds a cnpgCluster with the given hibernation annotation value.
// Pass "" to omit the annotation entirely.
func newTestCnpgCluster(hibernation string) *cnpgCluster {
	metadata := map[string]any{
		"name":      "test-cnpgcluster",
		"namespace": "default",
	}

	if hibernation != "" {
		metadata["annotations"] = map[string]any{
			cnpgHibernationAnnotation: hibernation,
		}
	}

	obj := map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata":   metadata,
		"spec": map[string]any{
			"instances": int64(3),
		},
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(cnpgClusterGVK)

	return &cnpgCluster{Unstructured: u}
}

func TestCnpgCluster_GetSuspend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		hibernation string
		wantCurrent values.Replicas
	}{
		{
			name:        "hibernation on is suspended",
			hibernation: cnpgHibernationOn,
			wantCurrent: values.BooleanReplicas(true),
		},
		{
			name:        "hibernation off is not suspended",
			hibernation: cnpgHibernationOff,
			wantCurrent: values.BooleanReplicas(false),
		},
		{
			name:        "absent annotation is not suspended",
			hibernation: "",
			wantCurrent: values.BooleanReplicas(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cluster := newTestCnpgCluster(test.hibernation)

			current, target := cluster.getSuspend()

			assert.Equal(t, test.wantCurrent, current)
			assert.Equal(t, values.BooleanReplicas(true), target)
		})
	}
}

func TestCnpgCluster_SetSuspend(t *testing.T) {
	t.Parallel()

	cluster := newTestCnpgCluster("")

	cluster.setSuspend(true)
	assert.Equal(t, cnpgHibernationOn, cluster.GetAnnotations()[cnpgHibernationAnnotation])

	current, _ := cluster.getSuspend()
	assert.Equal(t, values.BooleanReplicas(true), current)

	cluster.setSuspend(false)
	assert.Equal(t, cnpgHibernationOff, cluster.GetAnnotations()[cnpgHibernationAnnotation])

	current, _ = cluster.getSuspend()
	assert.Equal(t, values.BooleanReplicas(false), current)
}

func TestCnpgCluster_SetSuspend_PreservesOtherAnnotations(t *testing.T) {
	t.Parallel()

	cluster := newTestCnpgCluster("")
	cluster.SetAnnotations(map[string]string{"keep.me/here": "value"})

	cluster.setSuspend(true)

	annotations := cluster.GetAnnotations()
	assert.Equal(t, "value", annotations["keep.me/here"])
	assert.Equal(t, cnpgHibernationOn, annotations[cnpgHibernationAnnotation])
}

func TestCnpgCluster_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	cluster := newTestCnpgCluster("")
	cluster.Object["spec"].(map[string]any)["resources"] = map[string]any{
		"requests": map[string]any{
			"cpu":    "500m",
			"memory": "1Gi",
		},
	}

	saved := cluster.getSavedResourcesRequests()

	// 3 instances * 500m = 1.5 CPU, 3 * 1Gi memory.
	assert.InDelta(t, 1.5, saved.TotalCPU(), 0.0001)
	assert.InDelta(t, 3*1024*1024*1024, saved.TotalMemory(), 0.0001)
}

func TestCnpgCluster_GetSavedResourcesRequests_NoResources(t *testing.T) {
	t.Parallel()

	cluster := newTestCnpgCluster("")

	saved := cluster.getSavedResourcesRequests()

	assert.InDelta(t, 0, saved.TotalCPU(), 0.0001)
	assert.InDelta(t, 0, saved.TotalMemory(), 0.0001)
}

func TestCnpgCluster_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestCnpgCluster(cnpgHibernationOff)

	copied, err := original.Copy()
	require.NoError(t, err)

	ssw, ok := copied.(*suspendScaledWorkload)
	require.True(t, ok)

	copiedCluster, ok := ssw.suspendScaledResource.(*cnpgCluster)
	require.True(t, ok)

	// Mutating the copy must not touch the original.
	copiedCluster.setSuspend(true)

	assert.Equal(t, cnpgHibernationOn, copiedCluster.GetAnnotations()[cnpgHibernationAnnotation])
	assert.Equal(t, cnpgHibernationOff, original.GetAnnotations()[cnpgHibernationAnnotation])
}

func TestCnpgCluster_GroupVersionKind(t *testing.T) {
	t.Parallel()

	cluster := newTestCnpgCluster("")

	gvk := cluster.GroupVersionKind()
	assert.Equal(t, "postgresql.cnpg.io", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Cluster", gvk.Kind)
}
