package scalable

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// TestDaemonSetScaleUp tests the ScaleUp method of the daemonSet struct.
func TestDaemonSetScaleUp(t *testing.T) {
	ds := &daemonSet{
		DaemonSet: &appsv1.DaemonSet{
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{
							labelMatchNone: "true",
						},
					},
				},
			},
		},
	}

	err := ds.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, exists := ds.Spec.Template.Spec.NodeSelector[labelMatchNone]; exists {
		t.Errorf("expected label %s to be removed, but it still exists", labelMatchNone)
	}
}

// TestDaemonSetScaleDown tests the ScaleDown method of the daemonSet struct.
func TestDaemonSetScaleDown(t *testing.T) {
	ds := &daemonSet{
		DaemonSet: &appsv1.DaemonSet{
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						NodeSelector: map[string]string{},
					},
				},
			},
		},
	}

	err := ds.ScaleDown(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if val, exists := ds.Spec.Template.Spec.NodeSelector[labelMatchNone]; !exists || val != "true" {
		t.Errorf("expected label %s to be added with value 'true', but got value %v", labelMatchNone, val)
	}
}
