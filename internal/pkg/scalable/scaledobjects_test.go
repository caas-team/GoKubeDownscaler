package scalable

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestScaledObjectsScaleUp tests the ScaleUp method of the scaledObject struct.
func TestScaledObjectsScaleUp(t *testing.T) {
	// Mock a scaledObject with the paused annotation set
	so := &scaledObject{
		ScaledObject: &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotationKedaPausedReplicas: "2",
				},
			},
		},
	}

	// Mock original replicas to test ScaleUp
	originalReplicas := 2
	setOriginalReplicas(originalReplicas, so)

	err := so.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Ensure that the annotation was updated correctly
	if val, ok := so.Annotations[annotationKedaPausedReplicas]; ok {
		t.Errorf("expected annotation value to be %s, got %s", annotationKedaPausedReplicas, val)
	}
}

// TestScaledObjectsScaleDown tests the ScaleDown method of the scaledObject struct.
func TestScaledObjectsScaleDown(t *testing.T) {
	// Mock a scaledObject without the paused annotation initially
	so := &scaledObject{
		ScaledObject: &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
			},
		},
	}

	downscaleReplicas := 3

	err := so.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Ensure that the annotation was set correctly
	if val, ok := so.Annotations[annotationKedaPausedReplicas]; !ok {
		t.Errorf("expected annotation %s to be set, but it was not", annotationKedaPausedReplicas)
	} else if val != "3" {
		t.Errorf("expected annotation value to be %d, got %s", downscaleReplicas, val)
	}

	originalReplicas, _ := getOriginalReplicas(so)
	if originalReplicas == nil || *originalReplicas != values.Undefined {
		t.Errorf("expected original replicas to be undefined, got %v", originalReplicas)
	}
}

// TestScaledObjectsScaleDownWithExistingAnnotation tests the ScaleDown method of the scaledObject struct when the paused annotation is already present.
func TestScaledObjectsScaleDownWithExistingAnnotation(t *testing.T) {
	// Mock a scaledObject with the paused annotation already set to 5
	so := &scaledObject{
		ScaledObject: &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					annotationKedaPausedReplicas: "5",
				},
			},
		},
	}

	downscaleReplicas := 3

	err := so.ScaleDown(downscaleReplicas)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Ensure that the annotation was updated correctly
	if val, ok := so.Annotations[annotationKedaPausedReplicas]; !ok {
		t.Errorf("expected annotation %s to be set, but it was not", annotationKedaPausedReplicas)
	} else if val != "3" {
		t.Errorf("expected annotation value to be %d, got %s", downscaleReplicas, val)
	}

	originalReplicas, err := getOriginalReplicas(so)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if originalReplicas == nil || *originalReplicas != 5 {
		t.Errorf("expected original replicas to be 5, got %v", originalReplicas)
	}
}
