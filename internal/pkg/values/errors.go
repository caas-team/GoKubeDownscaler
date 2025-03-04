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

type IncompatibleValuesError struct {
	ErrorType string
	Message   string
}

func (i *IncompatibleValuesError) Error() string {
	return fmt.Sprintf("%s: %s", i.ErrorType, i.Message)
}

func NewIncompatibleValuesError(errorType string, msg string) error {
	return &IncompatibleValuesError{errorType, msg}
}

type MissingConfigurationError struct {
	ErrorType string
	Message   string
}

func (m *MissingConfigurationError) Error() string {
	return fmt.Sprintf("%s: %s", m.ErrorType, m.Message)
}

func NewMissingConfigurationError(errorType string, msg string) error {
	return &IncompatibleValuesError{errorType, msg}
}

// Defined error types with recognizable error code --> evaluate error codes for documentation
var (
	ForceUpAndDowntimeError  = "ForceUpAndDowntime #Errorcode 1"
	UpAndDowntime            = "UpAndDowntimeError #Errorcode 2"
	TimeAndPeriod            = "TimeAndPeriodError #Errorcode 3"
	InvalidDownscaleReplicas = "InvalidDownscaleReplicasError #Error 4"
	ValueNotSet              = "ValueNotSet #Errorcode 4"
	AnnotationNotSet         = "AnnotationNotSet #Errorcode 5"
)

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
		Value:   value,
	}
}
