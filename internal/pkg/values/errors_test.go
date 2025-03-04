package values

import (
	testing "testing"
)

// Runs the test for NewForceUpAndDownTimeError()-method.
func TestNewForceUpAndDownTimeError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(ForceUpAndDowntimeError, "both forceUptime and forceDowntime are defined")
	if err == nil {
		t.Error("NewForceUpAndDownTimeError()-method should have thrown an error")
	}
}

// Runs the test for NewUpAndDownTimeError()-method.
func TestNewUpAndDowntimeError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(UpAndDowntime, "both uptime and downtime are defined")
	if err == nil {
		t.Error("NewUpAndDownTimeError()-method should have thrown an error")
	}
}

// Runs the test for NewTimeAndPeriodError()-method.
func TestNewTimeAndPeriodError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(TimeAndPeriod, "")
	if err == nil {
		t.Error("NewTimeAndPeriodError()-method should have thrown an error")
	}
}

// Runs the test for NewInvalidDownscaleReplicasError()-method.
func TestNewInvalidDownscaleReplicasError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(InvalidDownscaleReplicas, "")
	if err == nil {
		t.Error("NewInvalidDownscaleReplicasError()-method should have thrown an error")
	}
}

// Runs the test for NewValueNotSetError()-method.
func TestNewValueNotSetError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(ValueNotSet, "")
	if err == nil {
		t.Error("NewValueNotSetError()-method should have thrown an error")
	}
}

// Runs the test for NewAnnotationsNotSetError()-method.
func TestNewAnnotationsNotSetError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(AnnotationNotSet, "")
	if err == nil {
		t.Error("NewAnnotationsNotSetError()-method should have thrown an error")
	}
}

// Runs the test for NewInvalidWeekdayError()-method.
func TestNewInvalidWeekdayError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(TimeAndPeriod, "")
	if err == nil {
		t.Error("NewInvalidWeekdayError()-method should have thrown an error")
	}
}

// Runs the test for NewInvalidRelativeTimespanError()-method.
func TestNewInvalidRelativeTimespanError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(TimeAndPeriod, "")
	if err == nil {
		t.Error("NewInvalidRelativeTimespanError()-method should have thrown an error")
	}
}

// Runs the test for NewTimeOfDataeOutOfRangeError()-method.
func TestNewTimeOfDateOutOfRangeError(t *testing.T) {
	t.Parallel()

	err := NewIncompatibleValuesError(TimeAndPeriod, "")
	if err == nil {
		t.Error("NewTimeOfDateOutOfRangeError()-method should have thrown an error")
	}
}
