package values

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	errForceUpAndDownTime       = errors.New("error: both forceUptime and forceDowntime are defined")
	errUpAndDownTime            = errors.New("error: both uptime and downtime are defined")
	errTimeAndPeriod            = errors.New("error: both a time and a period is defined")
	errInvalidDownscaleReplicas = errors.New("error: downscale replicas value is invalid")
	errValueNotSet              = errors.New("error: no layer implements this value")
	errAnnotationNotSet         = errors.New("error: annotation isn't set on workload")
)

const Undefined = -1 // Undefined represents an undefined integer value

// scaling is an enum that describes the current scaling
type scaling int

const (
	ScalingNone   scaling = iota // no scaling set in this layer, go to next layer
	ScalingIgnore                // not scaling
	ScalingDown                  // scaling down
	ScalingUp                    // scaling up
)

// NewLayer gets a new layer with the default values
func NewLayer() Layer {
	return Layer{
		DownscaleReplicas: Undefined,
		GracePeriod:       Undefined,
	}
}

// Layer represents a value Layer
type Layer struct {
	DownscalePeriod   timeSpans    // periods to downscale in
	DownTime          timeSpans    // within these timespans workloads will be scaled down, outside of them they will be scaled up
	UpscalePeriod     timeSpans    // periods to upscale in
	UpTime            timeSpans    // within these timespans workloads will be scaled up, outside of them they will be scaled down
	Exclude           triStateBool // if workload should be excluded
	ExcludeUntil      time.Time    // until when the workload should be excluded
	ForceUptime       triStateBool // force workload into a uptime state
	ForceDowntime     triStateBool // force workload into a downtime state
	DownscaleReplicas int32        // the replicas to scale down to
	GracePeriod       Duration     // grace period until new workloads will be scaled down
}

// isScalingExcluded checks if scaling is excluded, nil represents a not set state
func (l Layer) isScalingExcluded() *bool {
	if l.Exclude.isSet {
		return &l.Exclude.value
	}
	if ok := l.ExcludeUntil.After(time.Now()); ok {
		return &ok
	}
	return nil
}

// CheckForIncompatibleFields checks if there are incompatible fields
func (l Layer) CheckForIncompatibleFields() error {
	// force down and uptime
	if l.ForceDowntime.isSet &&
		l.ForceDowntime.value &&
		l.ForceUptime.isSet &&
		l.ForceUptime.value {
		return errForceUpAndDownTime
	}
	// downscale replicas invalid
	if l.DownscaleReplicas != Undefined && l.DownscaleReplicas < 0 {
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

// getCurrentScaling gets the current scaling, not checking for incompatibility
func (l Layer) getCurrentScaling() scaling {
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

// getForcedScaling checks if the layer has forced scaling enabled and returns the matching scaling
func (l Layer) getForcedScaling() scaling {
	var forcedScaling scaling
	if l.ForceDowntime.isSet && l.ForceDowntime.value {
		forcedScaling = ScalingDown
	}
	if l.ForceUptime.isSet && l.ForceUptime.value {
		forcedScaling = ScalingUp
	}
	return forcedScaling
}

type Layers []Layer

// GetCurrentScaling gets the current scaling of the first layer that implements scaling
func (l Layers) GetCurrentScaling() scaling {
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

// GetDownscaleReplicas gets the downscale replicas of the first layer that implements downscale replicas
func (l Layers) GetDownscaleReplicas() (int32, error) {
	for _, layer := range l {
		downscaleReplicas := layer.DownscaleReplicas
		if downscaleReplicas == Undefined {
			continue
		}

		return downscaleReplicas, nil
	}
	return 0, errValueNotSet
}

// GetExcluded checks if any layer excludes scaling
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

// IsInGracePeriod gets the grace period of the uppermost layer that has it set
func (l Layers) IsInGracePeriod(timeAnnotation string, workloadAnnotations map[string]string, creationTime time.Time, logEvent resourceLogger, ctx context.Context) (bool, error) {
	var gracePeriod Duration = Undefined
	for _, layer := range l {
		if layer.GracePeriod == Undefined {
			continue
		}
		gracePeriod = layer.GracePeriod
		break
	}
	if gracePeriod == Undefined {
		return false, nil
	}

	if timeAnnotation != "" {
		timeString, ok := workloadAnnotations[timeAnnotation]
		if !ok {
			logEvent.ErrorInvalidAnnotation(timeAnnotation, fmt.Sprintf("annotation %q not present on this workload", timeAnnotation), ctx)
			return false, errAnnotationNotSet
		}
		var err error
		creationTime, err = time.Parse(time.RFC3339, timeString)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(timeAnnotation, fmt.Sprintf("failed to parse %q annotation as RFC3339 timestamp: %s", timeAnnotation, err.Error()), ctx)
			return false, fmt.Errorf("failed to parse timestamp in annotation: %w", err)
		}
	}
	gracePeriodUntil := creationTime.Add(time.Duration(gracePeriod))
	return time.Now().Before(gracePeriodUntil), nil
}
