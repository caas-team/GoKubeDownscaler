package values

// ConfigurationErrors
var (
	errForceUpAndDownTime = &ConfigurationError{reason: "error: both forceUptime and forceDowntime are defined", caused: ""}
	errUpAndDownTime      = &ConfigurationError{reason: "error: both uptime and downtime are defined", caused: ""}
	errTimeAndPeriod      = &ConfigurationError{reason: "error: both a time and a period is defined", caused: ""}
	//errInvalidConfig      = &ConfigurationError{reason: "error: invalid configuration", caused: ""}
)

type ConfigurationError struct {
	reason string
	caused string
}

func (e *ConfigurationError) Error() string {
	return e.reason
}

// ValidationErrors
var (
	errInvalidDownscaleReplicas = &ValidationError{reason: "error: downscale replicas value is invalid", caused: ""}
	errInvalidWeekday           = &ValidationError{reason: "error: specified weekday is invalid", caused: ""}
	errRelativeTimespanInvalid  = &ValidationError{reason: "error: specified relative timespan is invalid", caused: ""}
	errTimeOfDayOutOfRange      = &ValidationError{reason: "error: the time of day has fields that are out of range", caused: ""}
	//errInvalidAnnotation        = &ValidationError{reason: "error: invalid annotation", caused: ""}
)

type ValidationError struct {
	reason string
	caused string
}

func (e *ValidationError) Error() string {
	return e.reason
}

// RuntimeErrors
var (
	errValueNotSet      = &RuntimeError{reason: "error: no layer implements this value", caused: ""}
	errAnnotationNotSet = &RuntimeError{reason: "error: annotation isn't set on workload", caused: ""}
	//errOperationFailed  = &RuntimeError{reason: "error: operation failed", caused: ""}
)

type RuntimeError struct {
	reason string
	caused string
}

func (e *RuntimeError) Error() string {
	return e.reason
}
