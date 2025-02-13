package values

import "fmt"

// Error is a custom error type that includes the original error, a message and a value.
type Error struct {
	Message string
	Value   any
}

// Error returns the error message.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Value)
}

//
// Error messaged based on the layer values.
//

// NewForceUpAndDownTimeError returns an error when both forceUptime and forceDowntime are defined.
func NewForceUpAndDownTimeError(value any) error {
	return &Error{
		Message: "both forceUptime and forceDowntime are defined",
		Value:   value,
	}
}

// NewUpAndDownTimeError returns an error when both uptime and downtime are defined.
func NewUpAndDownTimeError(value any) error {
	return &Error{
		Message: "both uptime and downtime are defined",
		Value:   value,
	}
}

// NewTimeAndPeriodError returns an error when both a time and a period is defined.
func NewTimeAndPeriodError(value any) error {
	return &Error{
		Message: "both a time and a period is defined",
		Value:   value,
	}
}

// NewInvalidDownscaleReplicasError returns an error when the downscale replicas value is invalid.
func NewInvalidDownscaleReplicasError(value any) error {
	return &Error{
		Message: "downscale replicas value is invalid",
		Value:   value,
	}
}

// NewValueNotSetError returns an error when no layer implements this value.
func NewValueNotSetError(value any) error {
	return &Error{
		Message: "no layer implements this value",
		Value:   value,
	}
}

// NewAnnotationsNotSetError returns an error when the annotation isn't set on workload.
func NewAnnotationsNotSetError(value any) error {
	return &Error{
		Message: "annotation isn't set on workload",
		Value:   value,
	}
}

//
//
//

// NewInvalidWeekdayError returns an error when the specific weekday is invalid.
func NewInvalidWeekdayError(value any) error {
	return &Error{
		Message: "specific weekday is invalid",
		Value:   value,
	}
}

// NewInvalidRelativeTimespanError returns an error when the specific relative timespan is invalid.
func NewInvalidRelativeTimespanError(value any) error {
	return &Error{
		Message: "specific relative timespan is invalid",
		Value:   value,
	}
}

// NewTimeOfDateOutOfRangeError returns an error when the time of day has fields that are out of range.
func NewTimeOfDateOutOfRangeError(value any) error {
	return &Error{
		Message: "the time of day has fields that are out of range",
	}
}

//
//
//

///
///
///
