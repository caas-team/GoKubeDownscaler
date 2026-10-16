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
func newTestCnpgCluster(hibernationVal string) *cnpgCluster {
	annotations := map[string]any{}
	if hibernationVal != "" {
		annotations[cnpgHibernationAnnotation] = hibernationVal
	}

	obj := map[string]any{
		"apiVersion": "postgresql.cnpg.io/v1",
		"kind":       "Cluster",
		"metadata": map[string]any{
			"name":        "test-cnpgcluster",
			"namespace":   "default",
			"annotations": annotations,
		},
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
		name          string
		hibernatedVal string
		wantCurrent   values.Replicas
	}{
		{
			name:          "hibernation on",
			hibernatedVal: "on",
			wantCurrent:   values.BooleanReplicas(true),
		},
		{
			name:          "hibernation off",
			hibernatedVal: "off",
			wantCurrent:   values.BooleanReplicas(false),
		},
		{
			name:          "annotation absent",
			hibernatedVal: "",
			wantCurrent:   values.BooleanReplicas(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestCnpgCluster(test.hibernatedVal)

			current, target := w.getSuspend()

			assert.Equal(t, test.wantCurrent, current)
			assert.Equal(t, values.BooleanReplicas(true), target)
		})
	}
}

func TestCnpgCluster_SetSuspend(t *testing.T) {
	t.Parallel()

	cluster := newTestCnpgCluster("")

	cluster.setSuspend(true)
	assert.Equal(t, cnpgHibernationValueOn, cluster.GetAnnotations()[cnpgHibernationAnnotation])

	cluster.setSuspend(false)
	assert.Equal(t, cnpgHibernationValueOff, cluster.GetAnnotations()[cnpgHibernationAnnotation])
}

func TestCnpgCluster_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestCnpgCluster("off")
	ssw := &suspendScaledWorkload{suspendScaledResource: original}

	copyWorkload, err := ssw.Copy()
	require.NoError(t, err)

	original.setSuspend(true)

	origCurrent, _ := original.getSuspend()
	assert.Equal(t, values.BooleanReplicas(true), origCurrent)

	copySSW, ok := copyWorkload.(*suspendScaledWorkload)
	require.True(t, ok)
	copyCluster, ok := copySSW.suspendScaledResource.(*cnpgCluster)
	require.True(t, ok)

	copyCurrent, _ := copyCluster.getSuspend()
	assert.Equal(t, values.BooleanReplicas(false), copyCurrent)
}

func TestCnpgCluster_GVK(t *testing.T) {
	t.Parallel()

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(cnpgClusterGVK)
	w := &cnpgCluster{Unstructured: u}

	assert.Equal(t, cnpgClusterGVK.Group, w.GroupVersionKind().Group)
	assert.Equal(t, cnpgClusterGVK.Version, w.GroupVersionKind().Version)
	assert.Equal(t, cnpgClusterGVK.Kind, w.GroupVersionKind().Kind)
}

func TestCnpgCluster_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	w := newTestCnpgCluster("off")
	require.NoError(t, unstructured.SetNestedField(w.Object, "500m", "spec", "resources", "requests", "cpu"))
	require.NoError(t, unstructured.SetNestedField(w.Object, "1Gi", "spec", "resources", "requests", "memory"))

	saved := w.getSavedResourcesRequests()

	// spec.instances is 3 in newTestCnpgCluster.
	assert.InDelta(t, 1.5, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 3*1024*1024*1024, saved.TotalMemory(), 1e5)
}

func TestCnpgCluster_GetSavedResourcesRequests_MissingResources(t *testing.T) {
	t.Parallel()

	w := newTestCnpgCluster("off")

	saved := w.getSavedResourcesRequests()

	assert.InDelta(t, 0, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 0, saved.TotalMemory(), 1e-9)
}
