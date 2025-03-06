package values

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
)

var (
	errForceUpAndDownTime       = errors.New("error: both forceUptime and forceDowntime are defined")
	errUpAndDownTime            = errors.New("error: both uptime and downtime are defined")
	errTimeAndPeriod            = errors.New("error: both a time and a period is defined")
	errInvalidDownscaleReplicas = errors.New("error: downscale replicas value is invalid")
	errValueNotSet              = errors.New("error: no scope implements this value")
	errAnnotationNotSet         = errors.New("error: annotation isn't set on workload")
)

// Scaling is an enum that describes the current Scaling.
type Scaling int

const (
	ScalingNone   Scaling = iota // no scaling set in this scope, go to next scope
	ScalingIgnore                // not scaling
	ScalingDown                  // scaling down
	ScalingUp                    // scaling up
)

// ScopeID is an enum that describes the current Scope.
type ScopeID int

const (
	ScopeWorkload    ScopeID = iota // identifies the scope present in the workload
	ScopeNamespace                  // identifies the scope present in the namespace
	ScopeCli                        // identifies the scope defined in the CLI
	ScopeEnvironment                // identifies the scope defined in the environment variables
)

// String gets the string representation of the ScopeID.
func (l ScopeID) String() string {
	return map[ScopeID]string{
		ScopeWorkload:    "ScopeWorkload",
		ScopeNamespace:   "ScopeNamespace",
		ScopeCli:         "ScopeCli",
		ScopeEnvironment: "ScopeEnvironment",
	}[l]
}

// NewScope gets a new scope with the default values.
func NewScope() Scope {
	return Scope{
		DownscaleReplicas: util.Undefined,
		GracePeriod:       util.Undefined,
	}
}

// Scope represents a value Scope.
type Scope struct {
	DownscalePeriod   timeSpans     // periods to downscale in
	DownTime          timeSpans     // within these timespans workloads will be scaled down, outside of them they will be scaled up
	UpscalePeriod     timeSpans     // periods to upscale in
	UpTime            timeSpans     // within these timespans workloads will be scaled up, outside of them they will be scaled down
	Exclude           triStateBool  // if workload should be excluded
	ExcludeUntil      time.Time     // until when the workload should be excluded
	ForceUptime       triStateBool  // force workload into an uptime state
	ForceDowntime     triStateBool  // force workload into a downtime state
	DownscaleReplicas int32         // the replicas to scale down to
	GracePeriod       time.Duration // grace period until new workloads will be scaled down
}

// isScalingExcluded checks if scaling is excluded, nil represents a not set state.
func (l *Scope) isScalingExcluded() *bool {
	if l.Exclude.isSet {
		return &l.Exclude.value
	}

	if ok := l.ExcludeUntil.After(time.Now()); ok {
		return &ok
	}

	return nil
}

// CheckForIncompatibleFields checks if there are incompatible fields.
func (l *Scope) CheckForIncompatibleFields() error { //nolint: cyclop // this is still fine to read, we could defnitly consider refactoring this in the future
	// force down and uptime
	if l.ForceDowntime.isSet &&
		l.ForceDowntime.value &&
		l.ForceUptime.isSet &&
		l.ForceUptime.value {
		return errForceUpAndDownTime
	}
	// downscale replicas invalid
	if l.DownscaleReplicas != util.Undefined && l.DownscaleReplicas < 0 {
		return errInvalidDownscaleReplicas
	}
	// up- and downtime
	if l.UpTime != nil && l.DownTime != nil {
		return errUpAndDownTime
	}
	// *time and *period
	if (l.UpTime != nil || l.DownTime != nil) &&
		(l.UpscalePeriod != nil || l.DownscalePeriod != nil) {
		return errTimeAndPeriod
	}

	return nil
}

// getCurrentScaling gets the current scaling, not checking for incompatibility.
func (l *Scope) getCurrentScaling() Scaling {
	// check times
	if l.DownTime != nil {
		if l.DownTime.inTimeSpans() {
			return ScalingDown
		}

		return ScalingUp
	}

	if l.UpTime != nil {
		if l.UpTime.inTimeSpans() {
			return ScalingUp
		}

		return ScalingDown
	}

	// check periods
	if l.DownscalePeriod != nil || l.UpscalePeriod != nil {
		if l.DownscalePeriod.inTimeSpans() {
			return ScalingDown
		}

		if l.UpscalePeriod.inTimeSpans() {
			return ScalingUp
		}

		return ScalingIgnore
	}

	return ScalingNone
}

// getForcedScaling checks if the scope has forced scaling enabled and returns the matching scaling.
func (l *Scope) getForcedScaling() Scaling {
	var forcedScaling Scaling

	if l.ForceDowntime.isSet && l.ForceDowntime.value {
		forcedScaling = ScalingDown
	}

	if l.ForceUptime.isSet && l.ForceUptime.value {
		forcedScaling = ScalingUp
	}

	return forcedScaling
}

type Scopes [4]*Scope

// GetCurrentScaling gets the current scaling of the first scope that implements scaling.
func (l Scopes) GetCurrentScaling() Scaling {
	// check for forced scaling
	for _, scope := range l {
		forcedScaling := scope.getForcedScaling()
		if forcedScaling != ScalingNone {
			return forcedScaling
		}
	}
	// check for time-based scaling
	for _, scope := range l {
		scopeScaling := scope.getCurrentScaling()
		if scopeScaling == ScalingNone {
			continue
		}

		return scopeScaling
	}

	return ScalingNone
}

// GetDownscaleReplicas gets the downscale replicas of the first scope that implements downscale replicas.
func (l Scopes) GetDownscaleReplicas() (int32, error) {
	for _, scope := range l {
		downscaleReplicas := scope.DownscaleReplicas
		if downscaleReplicas == util.Undefined {
			continue
		}

		return downscaleReplicas, nil
	}

	return 0, errValueNotSet
}

// GetExcluded checks if any scope excludes scaling.
func (l Scopes) GetExcluded() bool {
	for _, scope := range l {
		excluded := scope.isScalingExcluded()
		if excluded == nil {
			continue
		}

		return *excluded
	}

	return false
}

// IsInGracePeriod gets the grace period of the uppermost scope that has it set.
func (l Scopes) IsInGracePeriod(
	timeAnnotation string,
	workloadAnnotations map[string]string,
	creationTime time.Time,
	logEvent util.ResourceLogger,
	ctx context.Context,
) (bool, error) {
	var err error
	var gracePeriod time.Duration = util.Undefined

	for _, scope := range l {
		if scope.GracePeriod == util.Undefined {
			continue
		}

		gracePeriod = scope.GracePeriod

		break
	}

	if gracePeriod == util.Undefined {
		return false, nil
	}

	if timeAnnotation != "" {
		timeString, ok := workloadAnnotations[timeAnnotation]
		if !ok {
			logEvent.ErrorInvalidAnnotation(timeAnnotation, fmt.Sprintf("annotation %q not present on this workload", timeAnnotation), ctx)
			return false, errAnnotationNotSet
		}

		creationTime, err = time.Parse(time.RFC3339, timeString)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation as RFC3339 timestamp: %w", timeAnnotation, err)
			logEvent.ErrorInvalidAnnotation(timeAnnotation, err.Error(), ctx)

			return false, err
		}
	}

	gracePeriodUntil := creationTime.Add(gracePeriod)

	return time.Now().Before(gracePeriodUntil), nil
}

// String gets the string representation of the scopes.
func (l Scopes) String() string {
	var builder strings.Builder

	builder.WriteString("[")

	for i, scope := range l {
		if i > 0 {
			builder.WriteString(" ")
		}

		fmt.Fprintf(&builder, "%s:%+v", ScopeID(i), scope)
	}

	builder.WriteString("]")

	return builder.String()
}
