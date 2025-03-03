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
	errValueNotSet              = errors.New("error: no layer implements this value")
	errAnnotationNotSet         = errors.New("error: annotation isn't set on workload")
)

// Scaling is an enum that describes the current Scaling.
type Scaling int

const (
	ScalingNone   Scaling = iota // no scaling set in this layer, go to next layer
	ScalingIgnore                // not scaling
	ScalingDown                  // scaling down
	ScalingUp                    // scaling up
)

// LayerID is an enum that describes the current Layer.
type LayerID int

const (
	LayerWorkload    LayerID = iota // identifies the layer present in the workload
	LayerNamespace                  // identifies the layer present in the namespace
	LayerCli                        // identifies the layer defined in the CLI
	LayerEnvironment                // identifies the layer defined in the environment variables
	LayerDefault                    // identifier for the layer which holds all default values
)

// String gets the string representation of the LayerID.
func (l LayerID) String() string {
	return map[LayerID]string{
		LayerWorkload:    "LayerWorkload",
		LayerNamespace:   "LayerNamespace",
		LayerCli:         "LayerCli",
		LayerEnvironment: "LayerEnvironment",
		LayerDefault:     "LayerDefault",
	}[l]
}

// NewLayer gets a new layer with the default values.
func NewLayer() Layer {
	return Layer{
		DownscaleReplicas: util.Undefined,
		GracePeriod:       util.Undefined,
	}
}

// Layer represents a value Layer.
type Layer struct {
	DownscalePeriod   timeSpans     // periods to downscale in
	DownTime          timeSpans     // within these timespans workloads will be scaled down, outside of them they will be scaled up
	UpscalePeriod     timeSpans     // periods to upscale in
	UpTime            timeSpans     // within these timespans workloads will be scaled up, outside of them they will be scaled down
	Exclude           timeSpans     // if workload should be excluded
	ExcludeUntil      *time.Time    // until when the workload should be excluded
	ForceUptime       timeSpans     // force workload into a uptime state when in one of the timespans
	ForceDowntime     timeSpans     // force workload into a downtime state when in one of the timespans
	DownscaleReplicas int32         // the replicas to scale down to
	GracePeriod       time.Duration // grace period until new workloads will be scaled down
}

func GetDefaultLayer() *Layer {
	return &Layer{
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
func (l *Layer) CheckForIncompatibleFields() error { //nolint: cyclop // this is still fine to read, we could defnitly consider refactoring this in the future
	// force down and uptime
	if l.ForceDowntime != nil && l.ForceUptime != nil {
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
func (l *Layer) getCurrentScaling() Scaling {
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
		return l.getScalingFromPeriods()
	}

	return ScalingNone
}

func (l *Layer) getScalingFromPeriods() Scaling {
	inDowntime := l.DownscalePeriod.inTimeSpans()
	inUptime := l.UpscalePeriod.inTimeSpans()

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

func (l *Layer) getForceScaling() Scaling {
	// check forced scaling
	if l.ForceDowntime.inTimeSpans() {
		return ScalingDown
	}

	if l.ForceUptime.inTimeSpans() {
		return ScalingUp
	}

	if l.ForceDowntime != nil || l.ForceUptime != nil {
		return ScalingIgnore // default result to non-unset value to avoid falling through
	}

	return ScalingNone
}

type Layers [5]*Layer

// GetCurrentScaling gets the current scaling of the first layer that implements scaling.
func (l Layers) GetCurrentScaling() Scaling {
	var result Scaling

	for _, layer := range l {
		forcedScaling := layer.getForceScaling()
		if forcedScaling == ScalingNone {
			continue // layer doesnt implement forced scaling; falling through
		}

		if forcedScaling == ScalingIgnore {
			result = ScalingIgnore // default to ScalingIgnore instead of ScalingNone for correct log message
			break                  // break out since forced scaling is set, but just inactive
		}

		return forcedScaling
	}

	for _, layer := range l {
		layerScaling := layer.getCurrentScaling()
		if layerScaling == ScalingNone {
			continue // layer doesnt implement scaling; falling through
		}

		return layerScaling
	}

	return result
}

// GetDownscaleReplicas gets the downscale replicas of the first layer that implements downscale replicas.
func (l Layers) GetDownscaleReplicas() (int32, error) {
	for _, layer := range l {
		downscaleReplicas := layer.DownscaleReplicas
		if downscaleReplicas == util.Undefined {
			continue
		}

		return downscaleReplicas, nil
	}

	return 0, errValueNotSet
}

// GetExcluded checks if the layers exclude scaling.
func (l Layers) GetExcluded() bool {
	var exclude timeSpans
	var excludeUntil time.Time

	for _, layer := range l {
		if layer.Exclude != nil {
			exclude = layer.Exclude
			break
		}
	}

	for _, layer := range l {
		if layer.ExcludeUntil != nil {
			excludeUntil = *layer.ExcludeUntil
			break
		}
	}

	return exclude.inTimeSpans() || excludeUntil.After(time.Now())
}

// IsInGracePeriod gets the grace period of the uppermost layer that has it set.
func (l Layers) IsInGracePeriod(
	timeAnnotation string,
	workloadAnnotations map[string]string,
	creationTime time.Time,
	logEvent util.ResourceLogger,
	ctx context.Context,
) (bool, error) {
	var err error
	var gracePeriod time.Duration = util.Undefined

	for _, layer := range l {
		if layer.GracePeriod == util.Undefined {
			continue
		}

		gracePeriod = layer.GracePeriod

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

// String gets the string representation of the layers.
func (l Layers) String() string {
	var builder strings.Builder

	builder.WriteString("[")

	for i, layer := range l {
		if i > 0 {
			builder.WriteString(" ")
		}

		fmt.Fprintf(&builder, "%s:%+v", LayerID(i), layer)
	}

	builder.WriteString("]")

	return builder.String()
}
