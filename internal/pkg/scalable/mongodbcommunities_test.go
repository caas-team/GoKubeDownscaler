package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestMongoDBCommunity builds a mongoDBCommunity with spec.members set to the given raw value.
// Pass nil to omit spec.members entirely. withLastSuccessfulConfiguration controls whether the
// operator's lastSuccessfulConfiguration annotation is present on the object.
func newTestMongoDBCommunity(membersVal any, withLastSuccessfulConfiguration bool) *mongoDBCommunity {
	annotations := map[string]any{}
	if withLastSuccessfulConfiguration {
		annotations[mongoLastSuccessfulConfigurationAnnotation] = `{"members":3}`
	}

	obj := map[string]any{
		"apiVersion": "mongodbcommunity.mongodb.com/v1",
		"kind":       "MongoDBCommunity",
		"metadata": map[string]any{
			"name":        "test-mongodbcommunity",
			"namespace":   "default",
			"annotations": annotations,
		},
		"spec": map[string]any{},
	}

	if membersVal != nil {
		obj["spec"].(map[string]any)["members"] = membersVal
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(mongoDBCommunityGVK)

	return &mongoDBCommunity{Unstructured: u}
}

func TestMongoDBCommunity_GetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		membersVal   any
		wantReplicas values.Replicas
		wantErr      bool
	}{
		{
			name:         "float64 members (API-server JSON)",
			membersVal:   float64(3),
			wantReplicas: values.AbsoluteReplicas(3),
		},
		{
			name:         "int64 members",
			membersVal:   int64(5),
			wantReplicas: values.AbsoluteReplicas(5),
		},
		{
			name:       "absent spec.members",
			membersVal: nil,
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mongoCommunity := newTestMongoDBCommunity(test.membersVal, true)

			got, err := mongoCommunity.getReplicas()

			if test.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, got)
		})
	}
}

func TestMongoDBCommunity_SetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                            string
		initialMembers                  any
		withLastSuccessfulConfiguration bool
		target                          int32
		wantErr                         bool
		wantReplicas                    values.Replicas
	}{
		{
			name:                            "scale down allowed with annotation",
			initialMembers:                  float64(3),
			withLastSuccessfulConfiguration: true,
			target:                          0,
			wantReplicas:                    values.AbsoluteReplicas(0),
		},
		{
			// spec.members must be left untouched when the scale down is refused.
			name:                            "scale down refused without annotation",
			initialMembers:                  float64(3),
			withLastSuccessfulConfiguration: false,
			target:                          0,
			wantErr:                         true,
			wantReplicas:                    values.AbsoluteReplicas(3),
		},
		{
			name:                            "scale up always allowed without annotation",
			initialMembers:                  float64(0),
			withLastSuccessfulConfiguration: false,
			target:                          3,
			wantReplicas:                    values.AbsoluteReplicas(3),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mongoCommunity := newTestMongoDBCommunity(test.initialMembers, test.withLastSuccessfulConfiguration)

			err := mongoCommunity.setReplicas(test.target)

			if test.wantErr {
				var scalingRefusedErr *ScalingRefusedError
				require.ErrorAs(t, err, &scalingRefusedErr)
			} else {
				require.NoError(t, err)
			}

			got, err := mongoCommunity.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantReplicas, got)
		})
	}
}

func TestMongoDBCommunity_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestMongoDBCommunity(float64(5), true)
	rsw := &replicaScaledWorkload{replicaScaledResource: original}

	copyWorkload, err := rsw.Copy()
	require.NoError(t, err)

	require.NoError(t, original.setReplicas(0))

	origReplicas, err := original.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), origReplicas)

	copyRSW, ok := copyWorkload.(*replicaScaledWorkload)
	require.True(t, ok)
	copyMDB, ok := copyRSW.replicaScaledResource.(*mongoDBCommunity)
	require.True(t, ok)

	copyReplicas, err := copyMDB.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(5), copyReplicas)
}

func TestMongoDBCommunity_GVK(t *testing.T) {
	t.Parallel()

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(mongoDBCommunityGVK)
	mongoCommunity := &mongoDBCommunity{Unstructured: u}

	assert.Equal(t, mongoDBCommunityGVK.Group, mongoCommunity.GroupVersionKind().Group)
	assert.Equal(t, mongoDBCommunityGVK.Version, mongoCommunity.GroupVersionKind().Version)
	assert.Equal(t, mongoDBCommunityGVK.Kind, mongoCommunity.GroupVersionKind().Kind)
}

func TestMongoDBCommunity_MustScaleChildren(t *testing.T) {
	t.Parallel()

	mongoCommunity := newTestMongoDBCommunity(float64(3), true)
	assert.True(t, mongoCommunity.MustScaleChildren())

	rsw := &replicaScaledWorkload{replicaScaledResource: mongoCommunity}
	assert.True(t, rsw.MustScaleChildren())
	assert.True(t, MustScaleChildren(rsw))
}

func TestMongoDBCommunity_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	mongoCommunity := newTestMongoDBCommunity(float64(3), true)
	require.NoError(t, unstructured.SetNestedSlice(mongoCommunity.Object, []any{
		map[string]any{
			"name": "mongod",
			"resources": map[string]any{
				"requests": map[string]any{
					"cpu":    "500m",
					"memory": "1Gi",
				},
			},
		},
	}, "spec", "statefulSet", "spec", "template", "spec", "containers"))

	saved := mongoCommunity.getSavedResourcesRequests(2)

	assert.InDelta(t, 1.0, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 2*1024*1024*1024, saved.TotalMemory(), 1e5)
}

func TestMongoDBCommunity_GetSavedResourcesRequests_MissingContainers(t *testing.T) {
	t.Parallel()

	mongoCommunity := newTestMongoDBCommunity(float64(3), true)

	saved := mongoCommunity.getSavedResourcesRequests(2)

	assert.InDelta(t, 0, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 0, saved.TotalMemory(), 1e-9)
}
