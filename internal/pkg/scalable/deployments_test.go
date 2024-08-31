package scalable

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

// TestDeploymentsScaleUp tests the ScaleUp method of the deployment struct.
func TestDeploymentsScaleUp(t *testing.T) {
	originalReplicas := int32(5)
	d := &deployment{
		Deployment: &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &originalReplicas,
			},
		},
	}

	// Mock setting the original replicas
	setOriginalReplicas(int(originalReplicas), d)

	err := d.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *d.Spec.Replicas != originalReplicas {
		t.Errorf("expected replicas to be %d, got %d", originalReplicas, *d.Spec.Replicas)
	}
}

// TestDeploymentsScaleDown tests the ScaleDown method of the deployment struct.
func TestDeploymentsScaleDown(t *testing.T) {
	originalReplicas := int32(5)
	d := &deployment{
		Deployment: &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &originalReplicas,
			},
		},
	}

	downscaleReplicas := 2
	err := d.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *d.Spec.Replicas != int32(downscaleReplicas) {
		t.Errorf("expected replicas to be %d, got %d", downscaleReplicas, *d.Spec.Replicas)
	}

	// Verify that original replicas are stored correctly
	storedOriginalReplicas, err := getOriginalReplicas(d)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *storedOriginalReplicas != int(originalReplicas) {
		t.Errorf("expected original replicas to be %d, got %d", originalReplicas, *storedOriginalReplicas)
	}
}
