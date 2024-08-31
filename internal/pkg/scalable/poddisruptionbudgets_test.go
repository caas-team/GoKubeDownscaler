package scalable

import (
	"testing"

	appsv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestPodDisruptionBudgetScaleUpMaxUnavailable tests the ScaleUp method of the podDisruptionBudget struct when MaxUnavailable is used.
func TestPodDisruptionBudgetScaleUpMaxUnavailable(t *testing.T) {
	pdb := &podDisruptionBudget{
		PodDisruptionBudget: &appsv1.PodDisruptionBudget{
			Spec: appsv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &intstr.IntOrString{IntVal: 2, Type: intstr.Int},
			},
		},
	}

	// Mock original replicas to test ScaleUp
	originalReplicas := 5
	setOriginalReplicas(originalReplicas, pdb)

	err := pdb.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pdb.getMaxUnavailableInt() != originalReplicas {
		t.Errorf("expected MaxUnavailable to be %d, got %d", originalReplicas, pdb.getMaxUnavailableInt())
	}
}

// TestPodDisruptionBudgetScaleDownMaxUnavailable tests the ScaleDown method of the podDisruptionBudget struct when MaxUnavailable is used.
func TestPodDisruptionBudgetScaleDownMaxUnavailable(t *testing.T) {
	pdb := &podDisruptionBudget{
		PodDisruptionBudget: &appsv1.PodDisruptionBudget{
			Spec: appsv1.PodDisruptionBudgetSpec{
				MaxUnavailable: &intstr.IntOrString{IntVal: 5, Type: intstr.Int},
			},
		},
	}

	downscaleReplicas := 2

	err := pdb.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pdb.getMaxUnavailableInt() != downscaleReplicas {
		t.Errorf("expected MaxUnavailable to be %d, got %d", downscaleReplicas, pdb.getMaxUnavailableInt())
	}

	originalReplicas, err := getOriginalReplicas(pdb)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if originalReplicas == nil || *originalReplicas != 5 {
		t.Errorf("expected original replicas to be 5, got %v", originalReplicas)
	}
}

// TestPodDisruptionBudgetScaleUpWithMinAvailable tests the ScaleUp method of the podDisruptionBudget struct when MinAvailable is used.
func TestPodDisruptionBudgetScaleUpWithMinAvailable(t *testing.T) {
	pdb := &podDisruptionBudget{
		PodDisruptionBudget: &appsv1.PodDisruptionBudget{
			Spec: appsv1.PodDisruptionBudgetSpec{
				MinAvailable: &intstr.IntOrString{IntVal: 2, Type: intstr.Int},
			},
		},
	}

	// Mock original replicas to test ScaleUp
	originalReplicas := 5
	setOriginalReplicas(originalReplicas, pdb)

	err := pdb.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pdb.getMinAvailableInt() != originalReplicas {
		t.Errorf("expected MinAvailable to be %d, got %d", originalReplicas, pdb.getMinAvailableInt())
	}
}

// TestPodDisruptionBudgetScaleDownWithMinAvailable tests the ScaleDown method of the podDisruptionBudget struct when MinAvailable is used.
func TestPodDisruptionBudgetScaleDownWithMinAvailable(t *testing.T) {
	pdb := &podDisruptionBudget{
		PodDisruptionBudget: &appsv1.PodDisruptionBudget{
			Spec: appsv1.PodDisruptionBudgetSpec{
				MinAvailable: &intstr.IntOrString{IntVal: 5, Type: intstr.Int},
			},
		},
	}

	downscaleReplicas := 2

	err := pdb.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pdb.getMinAvailableInt() != downscaleReplicas {
		t.Errorf("expected MinAvailable to be %d, got %d", downscaleReplicas, pdb.getMinAvailableInt())
	}

	originalReplicas, err := getOriginalReplicas(pdb)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if originalReplicas == nil || *originalReplicas != 5 {
		t.Errorf("expected original replicas to be 5, got %v", originalReplicas)
	}
}
