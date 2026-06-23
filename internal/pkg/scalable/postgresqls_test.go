package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	appsv1 "k8s.io/api/apps/v1"
)

func strPtr(value string) *string { return &value }

func TestPostgresql_SetGetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		numberOfInstance int32
		setTo            int32
		wantAfterSet     values.Replicas
	}{
		{
			name:             "set replicas to zero from non-zero",
			numberOfInstance: 3,
			setTo:            0,
			wantAfterSet:     values.AbsoluteReplicas(0),
		},
		{
			name:             "set replicas to non-zero from zero value",
			numberOfInstance: 0,
			setTo:            5,
			wantAfterSet:     values.AbsoluteReplicas(5),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pgResource := &postgresql{&acidv1.Postgresql{}}
			pgResource.Spec.NumberOfInstances = test.numberOfInstance

			// getReplicas never errors for postgresql (NumberOfInstances is a value int32).
			gotInitial, err := pgResource.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, values.AbsoluteReplicas(test.numberOfInstance), gotInitial)

			err = pgResource.setReplicas(test.setTo)
			require.NoError(t, err)

			gotAfterSet, err := pgResource.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, test.wantAfterSet, gotAfterSet)
		})
	}
}

func TestPostgresql_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resources    *acidv1.Resources
		diffReplicas int32
		wantCPU      float64 // in cores
		wantMemory   float64 // in bytes
	}{
		{
			name:         "no resources block",
			resources:    nil,
			diffReplicas: 3,
			wantCPU:      0,
			wantMemory:   0,
		},
		{
			name: "cpu and memory set",
			resources: &acidv1.Resources{
				ResourceRequests: acidv1.ResourceDescription{
					CPU:    strPtr("500m"),
					Memory: strPtr("1Gi"),
				},
			},
			diffReplicas: 3,
			wantCPU:      1.5,                    // 3 × 0.5 cores
			wantMemory:   3 * 1024 * 1024 * 1024, // 3 × 1Gi
		},
		{
			name: "only cpu set, memory nil",
			resources: &acidv1.Resources{
				ResourceRequests: acidv1.ResourceDescription{
					CPU: strPtr("250m"),
				},
			},
			diffReplicas: 2,
			wantCPU:      0.5, // 2 × 0.25 cores
			wantMemory:   0,
		},
		{
			name: "unparseable cpu treated as zero",
			resources: &acidv1.Resources{
				ResourceRequests: acidv1.ResourceDescription{
					CPU:    strPtr("not-a-quantity"),
					Memory: strPtr("64Mi"),
				},
			},
			diffReplicas: 1,
			wantCPU:      0,
			wantMemory:   64 * 1024 * 1024,
		},
		{
			name: "unparseable memory treated as zero",
			resources: &acidv1.Resources{
				ResourceRequests: acidv1.ResourceDescription{
					CPU:    strPtr("250m"),
					Memory: strPtr("not-a-quantity"),
				},
			},
			diffReplicas: 1,
			wantCPU:      0.25,
			wantMemory:   0,
		},
		{
			name: "resources block present but requests unset",
			resources: &acidv1.Resources{
				ResourceRequests: acidv1.ResourceDescription{},
			},
			diffReplicas: 4,
			wantCPU:      0,
			wantMemory:   0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pgResource := &postgresql{&acidv1.Postgresql{}}
			pgResource.Spec.Resources = test.resources

			saved := pgResource.getSavedResourcesRequests(test.diffReplicas)

			assert.InDelta(t, test.wantCPU, saved.TotalCPU(), 0.0001)
			assert.InDelta(t, test.wantMemory, saved.TotalMemory(), 1e5)
		})
	}
}

func TestPostgresql_Copy(t *testing.T) {
	t.Parallel()

	original := &postgresql{&acidv1.Postgresql{}}
	original.Spec.NumberOfInstances = 5

	copied, err := original.Copy()
	require.NoError(t, err)

	// Mutating the original must not affect the copy (deep copy).
	original.Spec.NumberOfInstances = 0

	rsw, ok := copied.(*replicaScaledWorkload)
	require.True(t, ok)

	copiedPg, ok := rsw.replicaScaledResource.(*postgresql)
	require.True(t, ok)
	assert.Equal(t, int32(5), copiedPg.Spec.NumberOfInstances)
}

func TestPostgresql_Compare(t *testing.T) {
	t.Parallel()

	original := &postgresql{&acidv1.Postgresql{}}
	original.Spec.NumberOfInstances = 5

	copied, err := original.Copy()
	require.NoError(t, err)

	rsw, ok := copied.(*replicaScaledWorkload)
	require.True(t, ok)

	copiedPg, ok := rsw.replicaScaledResource.(*postgresql)
	require.True(t, ok)

	err = copiedPg.setReplicas(0)
	require.NoError(t, err)

	patch, err := original.Compare(copied)
	require.NoError(t, err)
	assert.NotEmpty(t, patch, "expected a diff between differing instance counts")

	// Comparing against a workload that is not a *postgresql must error.
	wrongType := &replicaScaledWorkload{&deployment{&appsv1.Deployment{}}}
	_, err = original.Compare(wrongType)

	var expectErr *ExpectTypeGotTypeError
	assert.ErrorAs(t, err, &expectErr)
}

func TestPostgresql_ScaleDownScaleUp(t *testing.T) {
	t.Parallel()

	pgResource := &acidv1.Postgresql{}
	pgResource.Spec.NumberOfInstances = 5
	pgResource.Spec.Resources = &acidv1.Resources{
		ResourceRequests: acidv1.ResourceDescription{
			CPU:    strPtr("100m"),
			Memory: strPtr("64Mi"),
		},
	}

	workload := &replicaScaledWorkload{&postgresql{pgResource}}

	// Scale down to zero.
	saved, updateNeeded, err := workload.ScaleDown(values.AbsoluteReplicas(0))
	require.NoError(t, err)
	assert.True(t, updateNeeded)

	gotReplicas, err := workload.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), gotReplicas)

	assert.InDelta(t, 0.5, saved.TotalCPU(), 0.0001)                     // 5 × 0.1 cores
	assert.InDelta(t, float64(5*64*1024*1024), saved.TotalMemory(), 1e5) // 5 × 64Mi

	// Original replicas were recorded.
	original, err := getOriginalReplicas(workload)
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(5), original)

	// Scale back up restores the original instance count.
	updateNeeded, err = workload.ScaleUp()
	require.NoError(t, err)
	assert.True(t, updateNeeded)

	gotReplicas, err = workload.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(5), gotReplicas)

	// The original-replicas annotation must be cleared after scaling up, so a second
	// ScaleUp would be a no-op.
	_, err = getOriginalReplicas(workload)

	var unsetErr *OriginalReplicasUnsetError
	assert.ErrorAs(t, err, &unsetErr)
}

func TestParsePostgresqlFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("valid postgresql is parsed and wrapped", func(t *testing.T) {
		t.Parallel()

		raw := []byte(`{"metadata":{"name":"test-db","namespace":"default"},"spec":{"numberOfInstances":3}}`)

		workload, err := parsePostgresqlFromBytes(raw)
		require.NoError(t, err)

		assert.Equal(t, "test-db", workload.GetName())
		assert.Equal(t, "default", workload.GetNamespace())

		replicas, err := workload.(*replicaScaledWorkload).getReplicas()
		require.NoError(t, err)
		assert.Equal(t, values.AbsoluteReplicas(3), replicas)
	})

	t.Run("invalid json returns an error", func(t *testing.T) {
		t.Parallel()

		_, err := parsePostgresqlFromBytes([]byte(`{not valid json`))
		require.Error(t, err)
	})
}
