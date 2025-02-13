package values

import (
	testing "testing"
)

// Runs the test for NewForceUpAndDownTimeError()-method
func TestNewForceUpAndDownTimeError(t *testing.T) {
	t.Parallel()
	err := NewForceUpAndDownTimeError(nil)
	if err == nil {
		t.Error("NewForceUpAndDownTimeError()-method should have thrown an error")
	}
}

// Runs the test for NewUpAndDownTimeError()-method
func TestNewUpAndDowntimeError(t *testing.T) {
	t.Parallel()
	err := NewUpAndDownTimeError(nil)
	if err == nil {
		t.Error("NewUpAndDownTimeError()-method should have thrown an error")
	}
}

// Runs the test for NewTimeAndPeriodError()-method
func TestNewTimeAndPeriodError(t *testing.T) {
	t.Parallel()
	err := NewTimeAndPeriodError(nil)
	if err == nil {
		t.Error("NewTimeAndPeriodError()-method should have thrown an error")
	}
}

// Runs the test for NewInvalidDownscaleReplicasError()-method
func TestNewInvalidDownscaleReplicasError(t *testing.T) {
	t.Parallel()
	err := NewInvalidDownscaleReplicasError(nil)
	if err == nil {
		t.Error("NewInvalidDownscaleReplicasError()-method should have thrown an error")
	}
}

// Runs the test for NewValueNotSetError()-method
func TestNewValueNotSetError(t *testing.T) {
	t.Parallel()
	err := NewValueNotSetError(nil)
	if err == nil {
		t.Error("NewValueNotSetError()-method should have thrown an error")
	}
}

// Runs the test for NewAnnotationsNotSetError()-method
func TestNewAnnotationsNotSetError(t *testing.T) {
	t.Parallel()
	err := NewAnnotationsNotSetError(nil)
	if err == nil {
		t.Error("NewAnnotationsNotSetError()-method should have thrown an error")
	}
}
