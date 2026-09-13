package values

import (
	"fmt"
	"strconv"
)

// ScalingValue contains the value that selected a scaling decision.
// Exactly one of TimeSpans or Bool is expected to be populated.
type ScalingValue struct {
	TimeSpans []TimeSpan
	Bool      *bool
}

// NewBooleanScalingValue creates a ScalingValue containing a boolean value.
func NewBooleanScalingValue(value bool) ScalingValue {
	return ScalingValue{Bool: &value}
}

// scalingValueFromTimeSpans creates a ScalingValue containing the given time spans.
func scalingValueFromTimeSpans(values ...timeSpans) ScalingValue {
	total := 0

	for _, value := range values {
		total += len(value)
	}

	result := ScalingValue{
		TimeSpans: make([]TimeSpan, 0, total),
	}

	for _, value := range values {
		result.TimeSpans = append(result.TimeSpans, value...)
	}

	return result
}

// forceScalingValue returns the scaling value for a scope's force downtime and uptime.
func forceScalingValue(scope *Scope) ScalingValue {
	return scalingValueFromTimeSpans(scope.ForceDowntime, scope.ForceUptime)
}

// scalingValue returns the scaling value for a scope's downtime and uptime.
func scalingValue(scope *Scope) ScalingValue {
	if scope.DownTime != nil {
		return scalingValueFromTimeSpans(scope.DownTime)
	}

	if scope.UpTime != nil {
		return scalingValueFromTimeSpans(scope.UpTime)
	}

	return scalingValueFromTimeSpans(
		scope.DownscalePeriod,
		scope.UpscalePeriod,
	)
}

func (v ScalingValue) String() string {
	if v.Bool != nil {
		return strconv.FormatBool(*v.Bool)
	}

	return fmt.Sprint(v.TimeSpans)
}
