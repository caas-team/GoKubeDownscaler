package values

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
)

// Scaling is an enum that describes the current Scaling.
type Scaling int

const (
	ScalingNone       Scaling = iota // no scaling set in this scope, go to next scope
	ScalingIgnore                    // not scaling
	ScalingDown                      // scaling down
	ScalingUp                        // scaling up
	ScalingMultiple                  // multiple scalings with same priority matched, this should be handled as an error
	ScalingIncomplete                // not enough information to perform scaling, e.g. due to timespan being incomplete
)

// ScopeID is an enum that describes the current Scope.
type ScopeID int

const (
	ScopeWorkload    ScopeID = iota // identifies the scope present in the workload
	ScopeNamespace                  // identifies the scope present in the namespace
	ScopeCli                        // identifies the scope defined in the CLI
	ScopeEnvironment                // identifies the scope defined in the environment variables
	ScopeDefault                    // identifier for the scope which holds all default values
)

// String gets the string representation of the ScopeID.
func (s ScopeID) String() string {
	return map[ScopeID]string{
		ScopeWorkload:    "ScopeWorkload",
		ScopeNamespace:   "ScopeNamespace",
		ScopeCli:         "ScopeCli",
		ScopeEnvironment: "ScopeEnvironment",
		ScopeDefault:     "ScopeDefault",
	}[s]
}

// NewScope gets a new scope with all values in an unset state.
func NewScope() *Scope {
	return &Scope{
		DownscaleReplicas: nil,
		GracePeriod:       util.Undefined,
	}
}

// Scope represents a value Scope.
type Scope struct {
	DownscalePeriod   timeSpans       // periods to downscale in
	DownTime          timeSpans       // within these timespans workloads will be scaled down, outside of them they will be scaled up
	UpscalePeriod     timeSpans       // periods to upscale in
	UpTime            timeSpans       // within these timespans workloads will be scaled up, outside of them they will be scaled down
	Exclude           timeSpans       // defines when the workload should be excluded
	ExcludeUntil      *time.Time      // until when the workload should be excluded
	ForceUptime       timeSpans       // force workload into an uptime state when in one of the timespans
	ForceDowntime     timeSpans       // force workload into a downtime state when in one of the timespans
	DownscaleReplicas Replicas        // the replicas to scale down to
	GracePeriod       time.Duration   // grace period until new workloads will be scaled down
	ScaleChildren     triStateBool    // ownerReference will immediately trigger scaling of children workloads, when applicable
	UpscaleExcluded   triStateBool    // excluded workloads will be upscaled
	DefaultTimezone   *time.Location  // default timezone to use when not specified in a timespan, defaults to nil
	DefaultWeekFrame  *util.WeekFrame // default week frame to use when not specified in a timespan, defaults to nil
}

func GetDefaultScope() *Scope {
	return &Scope{
		DownscalePeriod:   nil,
		DownTime:          nil,
		UpscalePeriod:     nil,
		UpTime:            nil,
		Exclude:           nil,
		ExcludeUntil:      nil,
		ForceUptime:       nil,
		ForceDowntime:     nil,
		DownscaleReplicas: AbsoluteReplicas(0),
		GracePeriod:       15 * time.Minute,
		ScaleChildren:     triStateBool{isSet: false, value: false},
		UpscaleExcluded:   triStateBool{isSet: false, value: false},
		DefaultTimezone:   nil,
		DefaultWeekFrame:  nil,
	}
}

// CheckForIncompatibleFields checks if there are incompatible fields.
func (s *Scope) CheckForIncompatibleFields() error {
	// up- and downtime
	if s.UpTime != nil && s.DownTime != nil {
		return newIncompatibalFieldsError("uptime", "downtime")
	}
	// *time and *period
	if (s.UpTime != nil || s.DownTime != nil) &&
		(s.UpscalePeriod != nil || s.DownscalePeriod != nil) {
		return newIncompatibalFieldsError("time", "period")
	}

	return nil
}

// getCurrentScaling gets the current scaling, not checking for incompatibility.
func (s *Scope) getCurrentScaling(scopes Scopes) Scaling {
	// check times
	if s.DownTime != nil {
		inTimeSpans, err := s.DownTime.inTimeSpans(scopes)
		if err != nil {
			return ScalingIncomplete
		}

		if inTimeSpans {
			return ScalingDown
		}

		return ScalingUp
	}

	if s.UpTime != nil {
		inTimeSpans, err := s.UpTime.inTimeSpans(scopes)
		if err != nil {
			return ScalingIncomplete
		}

		if inTimeSpans {
			return ScalingUp
		}

		return ScalingDown
	}

	// check periods
	if s.DownscalePeriod != nil || s.UpscalePeriod != nil {
		return s.getScalingFromPeriods(scopes)
	}

	return ScalingNone
}

func (s *Scope) getScalingFromPeriods(scopes Scopes) Scaling {
	inDowntime, errInDowntime := s.DownscalePeriod.inTimeSpans(scopes)
	if errInDowntime != nil {
		return ScalingIncomplete
	}

	inUptime, errInUptime := s.UpscalePeriod.inTimeSpans(scopes)
	if errInUptime != nil {
		return ScalingIncomplete
	}

	if inUptime && inDowntime {
		return ScalingMultiple // this prevents unintended behavior; in the future this should be handled while checking for conflicts
	}

	if inDowntime {
		return ScalingDown
	}

	if inUptime {
		return ScalingUp
	}

	return ScalingIgnore
}

func (s *Scope) getForceScaling(scopes Scopes) Scaling {
	forceDowntime, errForceDowntime := s.ForceDowntime.inTimeSpans(scopes)
	if errForceDowntime != nil {
		return ScalingIncomplete
	}

	forceUptime, errForceUptime := s.ForceUptime.inTimeSpans(scopes)
	if errForceUptime != nil {
		return ScalingIncomplete
	}

	if forceDowntime && forceUptime {
		return ScalingMultiple // this prevents unintended behavior; in the future this should be handled while checking for conflicts
	}

	if forceDowntime {
		return ScalingDown
	}

	if forceUptime {
		return ScalingUp
	}

	if s.ForceDowntime != nil || s.ForceUptime != nil {
		return ScalingIgnore // default result to non-unset value to avoid falling through
	}

	return ScalingNone
}

type Scopes [5]*Scope

func (s Scopes) GetDefaultTimeSpan() *time.Location {
	for _, scope := range s {
		defaultTimezone := scope.DefaultTimezone
		if defaultTimezone == nil {
			continue
		}

		return defaultTimezone
	}

	return nil
}

func (s Scopes) GetDefaultWeekdayFrom() *time.Weekday {
	for _, scope := range s {
		weekdayFrom := scope.DefaultWeekFrame.WeekdayFrom
		if weekdayFrom == nil {
			continue
		}

		return weekdayFrom
	}

	return nil
}

func (s Scopes) GetDefaultWeekdayTo() *time.Weekday {
	for _, scope := range s {
		weekdayTo := scope.DefaultWeekFrame.WeekdayTo
		if weekdayTo == nil {
			continue
		}

		return weekdayTo
	}

	return nil
}

// GetCurrentScaling gets the current scaling of the first scope that implements scaling.
func (s Scopes) GetCurrentScaling() Scaling {
	var result Scaling

	for _, scope := range s {
		forcedScaling := scope.getForceScaling(s)
		if forcedScaling == ScalingNone {
			continue // scope doesnt implement forced scaling; falling through
		}

		if forcedScaling == ScalingIgnore {
			result = ScalingIgnore // default to ScalingIgnore instead of ScalingNone for correct log message
			break                  // break out since forced scaling is set, but just inactive
		}

		return forcedScaling
	}

	for _, scope := range s {
		scopeScaling := scope.getCurrentScaling(s)
		if scopeScaling == ScalingNone {
			continue // scope doesnt implement scaling; falling through
		}

		return scopeScaling
	}

	return result
}

// GetDownscaleReplicas gets the downscale replicas of the first scope that implements downscale replicas.
func (s Scopes) GetDownscaleReplicas() (Replicas, error) {
	for _, scope := range s {
		downscaleReplicas := scope.DownscaleReplicas
		if downscaleReplicas == nil {
			continue
		}

		return downscaleReplicas, nil
	}

	return nil, newValueNotSetError("downscaleReplicas")
}

// GetScaleChildren gets the scale children of the first scope that implements scale children.
func (s Scopes) GetScaleChildren() bool {
	for _, scope := range s {
		if scope.ScaleChildren.isSet {
			return scope.ScaleChildren.value
		}
	}

	return false
}

// GetExcluded checks if the scopes exclude scaling.
func (s Scopes) GetExcluded(scopes Scopes) bool {
	for _, scope := range s {
		if scope.Exclude == nil {
			continue
		}

		exclude, err := scope.Exclude.inTimeSpans(scopes)
		if err != nil {
			return false
		}

		if exclude {
			return true
		}

		break
	}

	for _, scope := range s {
		if scope.ExcludeUntil == nil {
			continue
		}

		if scope.ExcludeUntil.After(time.Now()) {
			return true
		}

		break
	}

	return false
}

// GetUpscaleExcluded check if the scopes upscale excluded workloads.
func (s Scopes) GetUpscaleExcluded() bool {
	for _, scope := range s {
		if scope.UpscaleExcluded.isSet && scope.UpscaleExcluded.value {
			return true
		}
	}

	return false
}

// IsInGracePeriod gets the grace period of the uppermost scope that has it set.
func (s Scopes) IsInGracePeriod(
	timeAnnotation string,
	workloadAnnotations map[string]string,
	creationTime time.Time,
	logEvent util.ResourceLogger,
	ctx context.Context,
) (bool, error) {
	var gracePeriod time.Duration = util.Undefined

	for _, scope := range s {
		if scope.GracePeriod == util.Undefined {
			continue
		}

		gracePeriod = scope.GracePeriod

		break
	}

	if gracePeriod == util.Undefined {
		return false, nil
	}

	creationTime, err := getWorkloadCreationTime(timeAnnotation, workloadAnnotations, creationTime, logEvent, ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get the workloads creation time: %w", err)
	}

	gracePeriodUntil := creationTime.Add(gracePeriod)

	return time.Now().Before(gracePeriodUntil), nil
}

func getWorkloadCreationTime(
	annotation string,
	annotations map[string]string,
	creationTime time.Time,
	logEvent util.ResourceLogger,
	ctx context.Context,
) (time.Time, error) {
	timeString, ok := annotations[annotation]
	if !ok {
		return creationTime, nil
	}

	creationTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		err = fmt.Errorf("failed to parse %q annotation as RFC3339 timestamp: %w", annotation, err)
		logEvent.ErrorInvalidAnnotation(annotation, err.Error(), ctx)

		return time.Time{}, err
	}

	return creationTime, nil
}

// String gets the string representation of the scopes.
func (s Scopes) String() string {
	var builder strings.Builder

	builder.WriteString("[")

	for i, scope := range s {
		if i > 0 {
			builder.WriteString(" ")
		}

		fmt.Fprintf(&builder, "%s:%+v", ScopeID(i), scope)
	}

	builder.WriteString("]")

	return builder.String()
}
