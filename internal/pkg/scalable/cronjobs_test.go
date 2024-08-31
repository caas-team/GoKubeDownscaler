package scalable

import (
	"testing"

	batch "k8s.io/api/batch/v1"
)

// TestCronJobScaleUp tests the ScaleUp method of the cronJob struct.
func TestCronJobScaleUp(t *testing.T) {
	cj := &cronJob{
		CronJob: &batch.CronJob{
			Spec: batch.CronJobSpec{
				Suspend: new(bool),
			},
		},
	}

	*cj.Spec.Suspend = true
	err := cj.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *cj.Spec.Suspend {
		t.Errorf("expected Suspend to be false, got true")
	}
}

// TestCronJobScaleDown tests the ScaleDown method of the cronJob struct.
func TestCronJobScaleDown(t *testing.T) {
	cj := &cronJob{
		CronJob: &batch.CronJob{
			Spec: batch.CronJobSpec{
				Suspend: new(bool),
			},
		},
	}

	*cj.Spec.Suspend = false
	err := cj.ScaleDown(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !*cj.Spec.Suspend {
		t.Errorf("expected Suspend to be true, got false")
	}
}
