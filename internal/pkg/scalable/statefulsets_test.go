package scalable

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestScaleUp tests the ScaleUp method of the statefulSet struct.
func TestStatefulSetsScaleUp(t *testing.T) {
	// Mock a statefulSet with an initial replica count of 3
	originalReplicas := int32(3)
	ss := &statefulSet{
		StatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &originalReplicas,
			},
		},
	}

	// Mock original replicas to test ScaleUp
	setOriginalReplicas(int(originalReplicas), ss)

	err := ss.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Ensure that the replicas were restored to the original value
	if *ss.Spec.Replicas != originalReplicas {
		t.Errorf("expected replicas to be %d, got %d", originalReplicas, *ss.Spec.Replicas)
	}
}

// TestScaleDown tests the ScaleDown method of the statefulSet struct.
func TestStatefulSetsScaleDown(t *testing.T) {
	// Mock a statefulSet with an initial replica count of 5
	originalReplicas := int32(5)
	ss := &statefulSet{
		StatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &originalReplicas,
			},
		},
	}

	// Mock downscaling to 2 replicas
	downscaleReplicas := 2

	err := ss.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Ensure that the replicas were set to the downscaled value
	if *ss.Spec.Replicas != int32(downscaleReplicas) {
		t.Errorf("expected replicas to be %d, got %d", downscaleReplicas, *ss.Spec.Replicas)
	}

	// Ensure that the original replicas were stored correctly
	originalReplicasAfter, _ := getOriginalReplicas(ss)
	if originalReplicasAfter == nil || *originalReplicasAfter != int(originalReplicas) {
		t.Errorf("expected original replicas to be %d, got %v", originalReplicas, originalReplicasAfter)
	}
}
