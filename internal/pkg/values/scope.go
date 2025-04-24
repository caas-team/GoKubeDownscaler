package values

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
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

// NewScope gets a new scope with the default values.
func NewScope() *Scope {
	return &Scope{
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
	Exclude           timeSpans     // defines when the workload should be excluded
	ExcludeUntil      *time.Time    // until when the workload should be excluded
	ForceUptime       timeSpans     // force workload into an uptime state when in one of the timespans
	ForceDowntime     timeSpans     // force workload into a downtime state when in one of the timespans
	DownscaleReplicas int32         // the replicas to scale down to
	GracePeriod       time.Duration // grace period until new workloads will be scaled down
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
		DownscaleReplicas: 0,
		GracePeriod:       15 * time.Minute,
	}
}

// CheckForIncompatibleFields checks if there are incompatible fields.
func (s *Scope) CheckForIncompatibleFields() error { //nolint: cyclop // this is still fine to read, we could defnitly consider refactoring this in the future
	// force down and uptime
	if s.ForceDowntime != nil && s.ForceUptime != nil {
		return newIncompatibalFieldsError("forceUptime", "forceDowntime")
	}
	// downscale replicas invalid
	if s.DownscaleReplicas != util.Undefined && s.DownscaleReplicas < 0 {
		return newInvalidValueError(
			"downscale replicas has to be a positive integer",
			strconv.Itoa(int(s.DownscaleReplicas)),
		)
	}
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
func (s *Scope) getCurrentScaling() Scaling {
	// check times
	if s.DownTime != nil {
		if s.DownTime.inTimeSpans() {
			return ScalingDown
		}

		return ScalingUp
	}

	if s.UpTime != nil {
		if s.UpTime.inTimeSpans() {
			return ScalingUp
		}

		return ScalingDown
	}

	// check periods
	if s.DownscalePeriod != nil || s.UpscalePeriod != nil {
		return s.getScalingFromPeriods()
	}

	return ScalingNone
}

func (s *Scope) getScalingFromPeriods() Scaling {
	inDowntime := s.DownscalePeriod.inTimeSpans()
	inUptime := s.UpscalePeriod.inTimeSpans()

	if inUptime && inDowntime {
		return ScalingIgnore // this prevents unintended behavior; in the future this should be handled while checking for conflicts
	}

	if inDowntime {
		return ScalingDown
	}

	if inUptime {
		return ScalingUp
	}

	return ScalingIgnore
}

func (s *Scope) getForceScaling() Scaling {
	// check forced scaling
	if s.ForceDowntime.inTimeSpans() {
		return ScalingDown
	}

	if s.ForceUptime.inTimeSpans() {
		return ScalingUp
	}

	if s.ForceDowntime != nil || s.ForceUptime != nil {
		return ScalingIgnore // default result to non-unset value to avoid falling through
	}

	return ScalingNone
}

type Scopes [5]*Scope

// GetCurrentScaling gets the current scaling of the first scope that implements scaling.
func (s Scopes) GetCurrentScaling() Scaling {
	var result Scaling

	for _, scope := range s {
		forcedScaling := scope.getForceScaling()
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
		scopeScaling := scope.getCurrentScaling()
		if scopeScaling == ScalingNone {
			continue // scope doesnt implement scaling; falling through
		}

		return scopeScaling
	}

	return result
}

// GetDownscaleReplicas gets the downscale replicas of the first scope that implements downscale replicas.
func (s Scopes) GetDownscaleReplicas() (int32, error) {
	for _, scope := range s {
		downscaleReplicas := scope.DownscaleReplicas
		if downscaleReplicas == util.Undefined {
			continue
		}

		return downscaleReplicas, nil
	}

	return 0, newValueNotSetError("downscaleReplicas")
}

// GetExcluded checks if the scopes exclude scaling.
func (s Scopes) GetExcluded() bool {
	for _, scope := range s {
		if scope.Exclude == nil {
			continue
		}

		if scope.Exclude.inTimeSpans() {
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
