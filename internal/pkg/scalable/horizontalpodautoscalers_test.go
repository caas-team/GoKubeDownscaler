package scalable

import (
	"testing"

	appsv1 "k8s.io/api/autoscaling/v2"
)

// TestHPAScaleUp tests the ScaleUp method of the horizontalPodAutoscaler struct.
func TestHPAScaleUp(t *testing.T) {
	originalReplicas := int32(3)
	h := &horizontalPodAutoscaler{
		HorizontalPodAutoscaler: &appsv1.HorizontalPodAutoscaler{
			Spec: appsv1.HorizontalPodAutoscalerSpec{
				MinReplicas: &originalReplicas,
			},
		},
	}

	// Mock setting the original replicas
	setOriginalReplicas(int(originalReplicas), h)

	// Modify MinReplicas to simulate scaling down before scaling up
	newReplicas := int32(1)
	h.Spec.MinReplicas = &newReplicas

	err := h.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *h.Spec.MinReplicas != originalReplicas {
		t.Errorf("expected MinReplicas to be %d, got %d", originalReplicas, *h.Spec.MinReplicas)
	}
}

// TestHPAScaleDown tests the ScaleDown method of the horizontalPodAutoscaler struct.
func TestHPAScaleDown(t *testing.T) {
	originalReplicas := int32(5)
	h := &horizontalPodAutoscaler{
		HorizontalPodAutoscaler: &appsv1.HorizontalPodAutoscaler{
			Spec: appsv1.HorizontalPodAutoscalerSpec{
				MinReplicas: &originalReplicas,
			},
		},
	}

	downscaleReplicas := 2
	err := h.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *h.Spec.MinReplicas != int32(downscaleReplicas) {
		t.Errorf("expected MinReplicas to be %d, got %d", downscaleReplicas, *h.Spec.MinReplicas)
	}

	// Verify that original replicas are stored correctly
	storedOriginalReplicas, err := getOriginalReplicas(h)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *storedOriginalReplicas != int(originalReplicas) {
		t.Errorf("expected original replicas to be %d, got %d", originalReplicas, *storedOriginalReplicas)
	}
}
