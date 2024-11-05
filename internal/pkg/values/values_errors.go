package values

import (
	"errors"
)

// Errors for values package
var (
	errForceUpAndDownTime       = errors.New("error: both forceUptime and forceDowntime are defined")
	errUpAndDownTime            = errors.New("error: both uptime and downtime are defined")
	errTimeAndPeriod            = errors.New("error: both a time and a period is defined")
	errInvalidDownscaleReplicas = errors.New("error: downscale replicas value is invalid")
	errValueNotSet              = errors.New("error: no layer implements this value")
	errAnnotationNotSet         = errors.New("error: annotation isn't set on workload")
	errInvalidWeekday           = errors.New("error: specified weekday is invalid")
	errRelativeTimespanInvalid  = errors.New("error: specified relative timespan is invalid")
	errTimeOfDayOutOfRange      = errors.New("error: the time of day has fields that are out of rane")
)
