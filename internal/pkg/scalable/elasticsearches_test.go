//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestElasticsearch builds an elasticsearch with the given pause annotation.
// Pass "" to omit the annotation entirely.
func newTestElasticsearch(pause string) *elasticsearch {
	metadata := map[string]any{
		"name":      "test-es",
		"namespace": "default",
	}

	if pause != "" {
		metadata["annotations"] = map[string]any{eckPauseAnnotation: pause}
	}

	obj := map[string]any{
		"apiVersion": "elasticsearch.k8s.elastic.co/v1",
		"kind":       "Elasticsearch",
		"metadata":   metadata,
		"spec":       map[string]any{},
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(elasticsearchGVK)

	return &elasticsearch{Unstructured: u}
}

func TestElasticsearch_GetSuspend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pause       string
		wantCurrent values.Replicas
	}{
		{name: "pause true is suspended", pause: eckPauseOn, wantCurrent: values.BooleanReplicas(true)},
		{name: "pause false is not suspended", pause: eckPauseOff, wantCurrent: values.BooleanReplicas(false)},
		{name: "absent annotation is not suspended", pause: "", wantCurrent: values.BooleanReplicas(false)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cluster := newTestElasticsearch(test.pause)
			current, target := cluster.getSuspend()

			assert.Equal(t, test.wantCurrent, current)
			assert.Equal(t, values.BooleanReplicas(true), target)
		})
	}
}

func TestElasticsearch_SetSuspend(t *testing.T) {
	t.Parallel()

	cluster := newTestElasticsearch("")

	cluster.setSuspend(true)
	assert.Equal(t, eckPauseOn, cluster.GetAnnotations()[eckPauseAnnotation])

	cluster.setSuspend(false)
	assert.Equal(t, eckPauseOff, cluster.GetAnnotations()[eckPauseAnnotation])
}

// GetChildren depends on a live *kubernetes.Clientset (Clientsets.Kubernetes is
// the concrete type, not an interface — a fake clientset is not assignable), so
// like the sibling cronjob/advancedcronjob GetChildren it is not unit-tested
// here. Its behavior — a parked Elasticsearch's node-set StatefulSets scaled to
// 0 by label — is covered by the in-cluster Stage 1b gate against a real ECK.

func TestElasticsearch_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestElasticsearch(eckPauseOff)

	copied, err := original.Copy()
	require.NoError(t, err)

	ssw, ok := copied.(*suspendScaledWorkload)
	require.True(t, ok)

	copiedES, ok := ssw.suspendScaledResource.(*elasticsearch)
	require.True(t, ok)

	copiedES.setSuspend(true)

	assert.Equal(t, eckPauseOn, copiedES.GetAnnotations()[eckPauseAnnotation])
	assert.Equal(t, eckPauseOff, original.GetAnnotations()[eckPauseAnnotation])
}

func TestElasticsearch_GroupVersionKind(t *testing.T) {
	t.Parallel()

	cluster := newTestElasticsearch("")

	gvk := cluster.GroupVersionKind()
	assert.Equal(t, "elasticsearch.k8s.elastic.co", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Elasticsearch", gvk.Kind)
}
