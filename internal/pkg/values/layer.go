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
	Exclude           triStateBool  // if workload should be excluded
	ExcludeUntil      time.Time     // until when the workload should be excluded
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
		Exclude:           triStateBool{isSet: true, value: false},
		ExcludeUntil:      time.Time{},
		ForceUptime:       nil,
		ForceDowntime:     nil,
		DownscaleReplicas: 0,
		GracePeriod:       15 * time.Minute,
	}
}

// isScalingExcluded checks if scaling is excluded, nil represents a not set state.
func (l *Layer) isScalingExcluded() *bool {
	if l.Exclude.isSet && l.Exclude.value {
		return &l.Exclude.value
	}

	if ok := l.ExcludeUntil.After(time.Now()); ok {
		return &ok
	}

	return nil
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

// getForcedScaling checks if the layer has forced scaling enabled and returns the matching scaling.
func (l *Layer) getForcedScaling() Scaling {
	var forcedScaling Scaling

	if l.ForceDowntime.inTimeSpans() {
		forcedScaling = ScalingDown
	}

	if l.ForceUptime.inTimeSpans() {
		forcedScaling = ScalingUp
	}

	return forcedScaling
}

type Layers [5]*Layer

// GetCurrentScaling gets the current scaling of the first layer that implements scaling.
func (l Layers) GetCurrentScaling() Scaling {
	// check for forced scaling
	for _, layer := range l {
		forcedScaling := layer.getForcedScaling()
		if forcedScaling != ScalingNone {
			return forcedScaling
		}
	}
	// check for time-based scaling
	for _, layer := range l {
		layerScaling := layer.getCurrentScaling()
		if layerScaling == ScalingNone {
			continue
		}

		return layerScaling
	}

	return ScalingNone
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

// GetExcluded checks if any layer excludes scaling.
func (l Layers) GetExcluded() bool {
	for _, layer := range l {
		excluded := layer.isScalingExcluded()
		if excluded == nil {
			continue
		}

		return *excluded
	}

	return false
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
