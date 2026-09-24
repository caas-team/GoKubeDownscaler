package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestMongoDBCommunity builds a mongoDBCommunity with the given spec.members
// and an optional lastSuccessfulConfiguration annotation.
func newTestMongoDBCommunity(members any, withLastSuccessful bool) *mongoDBCommunity {
	metadata := map[string]any{
		"name":      "test-mongo",
		"namespace": "default",
	}

	if withLastSuccessful {
		metadata["annotations"] = map[string]any{
			mongoLastSuccessfulConfigurationAnnotation: "{}",
		}
	}

	spec := map[string]any{}
	if members != nil {
		spec["members"] = members
	}

	obj := map[string]any{
		"apiVersion": "mongodbcommunity.mongodb.com/v1",
		"kind":       "MongoDBCommunity",
		"metadata":   metadata,
		"spec":       spec,
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(mongoDBCommunityGVK)

	return &mongoDBCommunity{Unstructured: u}
}

func TestMongoDBCommunity_GetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		members    any
		wantResult values.Replicas
		wantErr    bool
	}{
		{name: "int64 members", members: int64(3), wantResult: values.AbsoluteReplicas(3)},
		{name: "float64 members (API JSON)", members: float64(5), wantResult: values.AbsoluteReplicas(5)},
		{name: "absent members", members: nil, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mongo := newTestMongoDBCommunity(test.members, true)
			got, err := mongo.getReplicas()

			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantResult, got)
		})
	}
}

// The load-bearing fork-only test: a MongoDBCommunity without the
// lastSuccessfulConfiguration annotation must be refused for park (members=0),
// but scaling up must always be allowed.
func TestMongoDBCommunity_SetReplicas_RefusesParkWithoutLastSuccessful(t *testing.T) {
	t.Parallel()

	t.Run("park refused when annotation absent", func(t *testing.T) {
		t.Parallel()

		mongo := newTestMongoDBCommunity(int64(3), false)
		err := mongo.setReplicas(0)
		require.Error(t, err, "must refuse to park a mongo with no known-good config")
		assert.Contains(t, err.Error(), mongoLastSuccessfulConfigurationAnnotation)
	})

	t.Run("park allowed when annotation present", func(t *testing.T) {
		t.Parallel()

		mongo := newTestMongoDBCommunity(int64(3), true)
		require.NoError(t, mongo.setReplicas(0))

		got, err := mongo.getReplicas()
		require.NoError(t, err)
		assert.Equal(t, values.AbsoluteReplicas(0), got)
	})

	t.Run("scale up always allowed even without annotation", func(t *testing.T) {
		t.Parallel()

		mongo := newTestMongoDBCommunity(int64(0), false)
		require.NoError(t, mongo.setReplicas(3), "scaling up must never be refused")
	})
}

func TestMongoDBCommunity_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestMongoDBCommunity(int64(3), true)

	copied, err := original.Copy()
	require.NoError(t, err)

	rsw, ok := copied.(*replicaScaledWorkload)
	require.True(t, ok)

	copiedMongo, ok := rsw.replicaScaledResource.(*mongoDBCommunity)
	require.True(t, ok)

	require.NoError(t, copiedMongo.setReplicas(0))

	gotCopy, err := copiedMongo.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), gotCopy)

	gotOrig, err := original.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(3), gotOrig, "mutating the copy must not touch the original")
}

func TestMongoDBCommunity_GroupVersionKind(t *testing.T) {
	t.Parallel()

	mongo := newTestMongoDBCommunity(int64(3), true)

	gvk := mongo.GroupVersionKind()
	assert.Equal(t, "mongodbcommunity.mongodb.com", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "MongoDBCommunity", gvk.Kind)
}
