package values

import "fmt"

// Error is a custom error type that includes the original error, a message and a value.
type Error struct {
	Message string
	Value   interface{}
}

// Error returns the error message.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Value)
}

//
// Error messaged based on the layer values.
//

// NewForceUpAndDownTimeError returns an error when both forceUptime and forceDowntime are defined.
func NewForceUpAndDownTimeError(value interface{}) error {
	return &Error{
		Message: "both forceUptime and forceDowntime are defined",
		Value:   value,
	}
}

// NewUpAndDownTimeError returns an error when both uptime and downtime are defined.
func NewUpAndDownTimeError(value interface{}) error {
	return &Error{
		Message: "both uptime and downtime are defined",
		Value:   value,
	}
}

// NewTimeAndPeriodError returns an error when both a time and a period is defined.
func NewTimeAndPeriodError(value interface{}) error {
	return &Error{
		Message: "both a time and a period is defined",
		Value:   value,
	}
}

// NewInvalidDownscaleReplicasError returns an error when the downscale replicas value is invalid.
func NewInvalidDownscaleReplicasError(value interface{}) error {
	return &Error{
		Message: "downscale replicas value is invalid",
		Value:   value,
	}
}

// NewValueNotSetError returns an error when no layer implements this value.
func NewValueNotSetError(value interface{}) error {
	return &Error{
		Message: "no layer implements this value",
		Value:   value,
	}
}

// NewAnnotationsNotSetError returns an error when the annotation isn't set on workload.
func NewAnnotationsNotSetError(value interface{}) error {
	return &Error{
		Message: "annotation isn't set on workload",
		Value:   value,
	}
}

//
//
//

//
//
//
