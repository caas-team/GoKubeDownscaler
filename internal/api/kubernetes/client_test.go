package kubernetes

import (
	"context"
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const annotationOriginalReplicas = "downscaler/original-replicas"

func newClientWithDeployment(t *testing.T, dep *appsv1.Deployment) (client, scalable.Workload) {
	t.Helper()

	fakeKube := fake.NewClientset(dep)
	c := client{
		clientsets: &scalable.Clientsets{Kubernetes: fakeKube},
	}

	workloads, err := scalable.GetWorkloads("deployments", dep.Namespace, c.clientsets, context.Background())
	require.NoError(t, err)
	require.Len(t, workloads, 1)

	return c, workloads[0]
}

func countUpdateActions(c client) int {
	count := 0
	for _, action := range c.clientsets.Kubernetes.(*fake.Clientset).Actions() {
		if action.GetVerb() == "update" {
			count++
		}
	}

	return count
}

func TestDownscaleWorkload_SkipsUpdateWhenAlreadyDownscaled(t *testing.T) {
	t.Parallel()

	zero := int32(0)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "already-down",
			Namespace:   "ns",
			Annotations: map[string]string{annotationOriginalReplicas: "5"},
		},
		Spec: appsv1.DeploymentSpec{Replicas: &zero},
	}

	c, workload := newClientWithDeployment(t, dep)

	_, err := c.DownscaleWorkload(values.AbsoluteReplicas(0), workload, context.Background())
	require.NoError(t, err)

	assert.Equal(t, 0, countUpdateActions(c), "Update API should not be called when workload is already in desired state")
}

func TestDownscaleWorkload_CallsUpdateWhenScalingRequired(t *testing.T) {
	t.Parallel()

	five := int32(5)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "to-down", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &five},
	}

	c, workload := newClientWithDeployment(t, dep)

	_, err := c.DownscaleWorkload(values.AbsoluteReplicas(0), workload, context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, countUpdateActions(c), "Update API should be called when replicas change")
}

func TestUpscaleWorkload_SkipsUpdateWhenNoOriginalReplicasAnnotation(t *testing.T) {
	t.Parallel()

	five := int32(5)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "already-up", Namespace: "ns"},
		Spec:       appsv1.DeploymentSpec{Replicas: &five},
	}

	c, workload := newClientWithDeployment(t, dep)

	require.NoError(t, c.UpscaleWorkload(workload, context.Background()))

	assert.Equal(t, 0, countUpdateActions(c), "Update API should not be called when there is nothing to restore")
}

func TestUpscaleWorkload_CallsUpdateWhenRestoringReplicas(t *testing.T) {
	t.Parallel()

	zero := int32(0)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "to-up",
			Namespace:   "ns",
			Annotations: map[string]string{annotationOriginalReplicas: "5"},
		},
		Spec: appsv1.DeploymentSpec{Replicas: &zero},
	}

	c, workload := newClientWithDeployment(t, dep)

	require.NoError(t, c.UpscaleWorkload(workload, context.Background()))

	assert.Equal(t, 1, countUpdateActions(c), "Update API should be called when restoring replicas")
}
