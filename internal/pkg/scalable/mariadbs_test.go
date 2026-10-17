package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// newTestMariaDB builds a mariaDB with spec.replicas set to the given raw value (nil omits it).
// haField, when non-empty, is set to {enabled: true} under spec (e.g. "galera" or "replication").
func newTestMariaDB(replicasVal any, haField string) *mariaDB {
	obj := map[string]any{
		"apiVersion": "k8s.mariadb.com/v1alpha1",
		"kind":       "MariaDB",
		"metadata": map[string]any{
			"name":      "test-mariadb",
			"namespace": "default",
		},
		"spec": map[string]any{},
	}

	spec := obj["spec"].(map[string]any)
	if replicasVal != nil {
		spec["replicas"] = replicasVal
	}

	if haField != "" {
		spec[haField] = map[string]any{"enabled": true}
	}

	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(mariaDBGVK)

	return &mariaDB{Unstructured: u}
}

func TestMariaDB_GetReplicas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		replicasVal  any
		wantReplicas values.Replicas
		wantErr      bool
	}{
		{
			name:         "float64 replicas (API-server JSON)",
			replicasVal:  float64(1),
			wantReplicas: values.AbsoluteReplicas(1),
		},
		{
			name:         "int64 replicas",
			replicasVal:  int64(3),
			wantReplicas: values.AbsoluteReplicas(3),
		},
		{
			name:         "absent spec.replicas falls back to CRD default",
			replicasVal:  nil,
			wantReplicas: values.AbsoluteReplicas(1),
		},
		{
			name:        "unexpected type",
			replicasVal: "three",
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestMariaDB(test.replicasVal, "")

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

func TestMariaDB_SetReplicas_ScaleDownStandalone(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(1), "")

	require.NoError(t, w.setReplicas(0))

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), got)
}

func TestMariaDB_SetReplicas_RefusedBelowHAMinimum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		haField string
		target  int32
	}{
		{name: "replication scale down", haField: "replication", target: 0},
		{name: "replication to 1", haField: "replication", target: 1},
		{name: "galera to 1", haField: "galera", target: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestMariaDB(float64(3), test.haField)

			err := w.setReplicas(test.target)

			var scalingRefusedErr *ScalingRefusedError
			require.ErrorAs(t, err, &scalingRefusedErr)

			got, err := w.getReplicas()
			require.NoError(t, err)
			assert.Equal(t, values.AbsoluteReplicas(3), got)
			assert.False(t, w.isSuspended())
		})
	}
}

func TestMariaDB_SetReplicas_GaleraScaleDownSuspends(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(3), "galera")

	require.NoError(t, w.setReplicas(0))

	assert.True(t, w.isSuspended())

	specReplicas, _, err := unstructured.NestedFieldNoCopy(w.Object, "spec", "replicas")
	require.NoError(t, err)
	assert.InDelta(t, 3, specReplicas, 0, "spec.replicas must stay valid for the operator webhook")

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(0), got)
}

func TestMariaDB_SetReplicas_GaleraWakeResumes(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(3), "galera")
	require.NoError(t, w.setReplicas(0))

	require.NoError(t, w.setReplicas(3))

	assert.False(t, w.isSuspended())

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(3), got)
}

func TestMariaDB_SetReplicas_GaleraScaleDownRefusedWithoutRecovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		recovery map[string]any
	}{
		{name: "recovery explicitly disabled", recovery: map[string]any{"enabled": false}},
		{name: "recovery block without enabled", recovery: map[string]any{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w := newTestMariaDB(float64(3), "galera")
			require.NoError(t, unstructured.SetNestedMap(w.Object, test.recovery, "spec", "galera", "recovery"))

			err := w.setReplicas(0)

			var scalingRefusedErr *ScalingRefusedError
			require.ErrorAs(t, err, &scalingRefusedErr)
			assert.False(t, w.isSuspended())
		})
	}
}

func TestMariaDB_SetReplicas_GaleraScaleDownAllowedWithRecoveryEnabled(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(3), "galera")
	require.NoError(t, unstructured.SetNestedField(w.Object, true, "spec", "galera", "recovery", "enabled"))

	require.NoError(t, w.setReplicas(0))
	assert.True(t, w.isSuspended())
}

func TestMariaDB_GaleraScaleDownThenUp(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(3), "galera")
	rsw := &replicaScaledWorkload{replicaScaledResource: w}

	down, err := rsw.ScaleDown(values.AbsoluteReplicas(0), nil)
	require.NoError(t, err)
	assert.True(t, down.IsUpdateNeeded)
	assert.True(t, w.isSuspended())
	assert.Equal(t, "3", w.GetAnnotations()[annotationOriginalReplicas])

	up, err := rsw.ScaleUp(nil)
	require.NoError(t, err)
	assert.True(t, up.IsUpdateNeeded)
	assert.False(t, w.isSuspended())
	assert.NotContains(t, w.GetAnnotations(), annotationOriginalReplicas)

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(3), got)
}

func TestMariaDB_GetReplicas_SuspendIgnoredWithoutGalera(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(1), "")
	require.NoError(t, unstructured.SetNestedField(w.Object, true, "spec", "suspend"))

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(1), got)
}

func TestMariaDB_SetReplicas_ScaleDownAllowedWhenHADisabled(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(1), "")
	require.NoError(t, unstructured.SetNestedField(w.Object, false, "spec", "galera", "enabled"))

	require.NoError(t, w.setReplicas(0))
}

func TestMariaDB_SetReplicas_ScaleUpAlwaysAllowed(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(0), "")

	require.NoError(t, w.setReplicas(1))

	got, err := w.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(1), got)
}

func TestMariaDB_Copy_IsDeepCopy(t *testing.T) {
	t.Parallel()

	original := newTestMariaDB(float64(1), "")
	rsw := &replicaScaledWorkload{replicaScaledResource: original}

	copyWorkload, err := rsw.Copy()
	require.NoError(t, err)

	require.NoError(t, original.setReplicas(0))

	copyRSW, ok := copyWorkload.(*replicaScaledWorkload)
	require.True(t, ok)
	copyMDB, ok := copyRSW.replicaScaledResource.(*mariaDB)
	require.True(t, ok)

	copyReplicas, err := copyMDB.getReplicas()
	require.NoError(t, err)
	assert.Equal(t, values.AbsoluteReplicas(1), copyReplicas)
}

func TestMariaDB_GVK(t *testing.T) {
	t.Parallel()

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(mariaDBGVK)
	w := &mariaDB{Unstructured: u}

	assert.Equal(t, "k8s.mariadb.com", w.GroupVersionKind().Group)
	assert.Equal(t, "v1alpha1", w.GroupVersionKind().Version)
	assert.Equal(t, "MariaDB", w.GroupVersionKind().Kind)
}

func TestMariaDB_GetSavedResourcesRequests(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(1), "")
	require.NoError(t, unstructured.SetNestedStringMap(w.Object, map[string]string{
		"cpu":    "500m",
		"memory": "1Gi",
	}, "spec", "resources", "requests"))

	saved := w.getSavedResourcesRequests(2)

	assert.InDelta(t, 1.0, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 2*1024*1024*1024, saved.TotalMemory(), 1e5)
}

func TestMariaDB_GetSavedResourcesRequests_MissingResources(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(1), "")

	saved := w.getSavedResourcesRequests(1)

	assert.InDelta(t, 0, saved.TotalCPU(), 1e-9)
	assert.InDelta(t, 0, saved.TotalMemory(), 1e-9)
}

func newTestDeploymentOwnedBy(name, ownerKind, ownerName string) appsv1.Deployment {
	dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}}
	if ownerKind != "" {
		dep.OwnerReferences = []metav1.OwnerReference{{Kind: ownerKind, Name: ownerName}}
	}

	return dep
}

func TestMariaDB_OwnedDeployments(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(1), "")

	children := w.ownedDeployments([]appsv1.Deployment{
		newTestDeploymentOwnedBy("test-mariadb-metrics", "MariaDB", "test-mariadb"),
		newTestDeploymentOwnedBy("other-mariadb-metrics", "MariaDB", "other-mariadb"),
		newTestDeploymentOwnedBy("same-name-other-kind", "MaxScale", "test-mariadb"),
		newTestDeploymentOwnedBy("unowned", "", ""),
	})

	require.Len(t, children, 1)
	assert.Equal(t, "test-mariadb-metrics", children[0].GetName())
	assert.Equal(t, deploymentKind, children[0].GroupVersionKind().Kind)
}

func TestMariaDB_MustScaleChildren(t *testing.T) {
	t.Parallel()

	rsw := &replicaScaledWorkload{replicaScaledResource: newTestMariaDB(float64(1), "")}
	assert.True(t, MustScaleChildren(rsw))
}

func TestMariaDB_OwnedStatefulSets(t *testing.T) {
	t.Parallel()

	w := newTestMariaDB(float64(3), "galera")

	owned := appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name: "test-mariadb", Namespace: "default",
		OwnerReferences: []metav1.OwnerReference{{Kind: "MariaDB", Name: "test-mariadb"}},
	}}
	other := appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{
		Name: "other", Namespace: "default",
		OwnerReferences: []metav1.OwnerReference{{Kind: "MariaDB", Name: "other-mariadb"}},
	}}

	children := w.ownedStatefulSets([]appsv1.StatefulSet{owned, other})

	require.Len(t, children, 1)
	assert.Equal(t, "test-mariadb", children[0].GetName())
	assert.Equal(t, statefulSetKind, children[0].GroupVersionKind().Kind)
}
