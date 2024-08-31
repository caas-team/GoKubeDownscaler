package scalable

import (
	"testing"

	batch "k8s.io/api/batch/v1"
)

// TestJobsScaleUp tests the ScaleUp method of the job struct.
func TestJobsScaleUp(t *testing.T) {
	j := &job{
		Job: &batch.Job{
			Spec: batch.JobSpec{
				Suspend: new(bool),
			},
		},
	}

	*j.Spec.Suspend = true // Initially, the job is suspended
	err := j.ScaleUp()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if *j.Spec.Suspend {
		t.Errorf("expected Suspend to be false, got true")
	}
}

// TestJobsScaleDown tests the ScaleDown method of the job struct.
func TestJobsScaleDown(t *testing.T) {
	j := &job{
		Job: &batch.Job{
			Spec: batch.JobSpec{
				Suspend: new(bool),
			},
		},
	}

	*j.Spec.Suspend = false // Initially, the job is not suspended
	err := j.ScaleDown(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !*j.Spec.Suspend {
		t.Errorf("expected Suspend to be true, got false")
	}
}
