package values

import (
	"errors"
	"fmt"
	"time"
)

var (
	errMaxGracePeriodExceeded   = errors.New("error: gracePeriod is longer than allowed by maxGracePeriod")
	errForceUpAndDownTime       = errors.New("error: both forceUptime and forceDowntime are defined")
	errUpAndDownTime            = errors.New("error: both uptime and downtime are defined")
	errTimeAndPeriod            = errors.New("error: both a time and a period is defined")
	errInvalidDownscaleReplicas = errors.New("error: downscale replicas value is invalid")
	errNoScalingProvided        = errors.New("error: no layer provided scaling")
)

type scaling int

const (
	ScalingNone         scaling = iota // no scaling set in this layer, go to next layer
	ScalingIncompatible                // incompatible scaling fields set, error
	ScalingIgnore                      // not scaling
	ScalingDown                        // scaling down
	ScalingUp                          // scaling up
)

type Layer struct {
	DownscalePeriod   TimeSpans // periods to downscale in
	DownTime          TimeSpans // within these timespans workloads will be scaled down, outside of them they will be scaled up
	UpscalePeriod     TimeSpans // periods to upscale in
	UpTime            TimeSpans // within these timespans workloads will be scaled up, outside of them they will be scaled down
	Exclude           bool      // if workload should be excluded
	ExcludeUntil      time.Time // until when the workload should be excluded
	ForceUptime       bool      // force workload into a uptime state
	ForceDowntime     bool      // force workload into a downtime state
	DownscaleReplicas int       // the replicas to scale down to
	GracePeriod       Duration  // grace period until new deployments will be scaled down
	TimeAnnotation    string    // annotation to use for grace-period instead of creation time
}

// isScalingExcluded checks if scaling is excluded
func (l Layer) isScalingExcluded() bool {
	return l.Exclude || l.ExcludeUntil.After(time.Now())
}

// checkForIncompatibleFields checks if there are incompatible fields
func (l Layer) checkForIncompatibleFields(maxGracePeriod time.Duration) error {
	// force down and uptime
	if l.ForceDowntime && l.ForceUptime {
		return errForceUpAndDownTime
	}
	// downscale replicas invalid
	if l.DownscaleReplicas <= 0 {
		return errInvalidDownscaleReplicas
	}
	// grace period > max grace period
	if maxGracePeriod != 0 && time.Duration(l.GracePeriod) > maxGracePeriod {
		return errMaxGracePeriodExceeded
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
	// check overwrites
	if l.isScalingExcluded() {
		return ScalingIgnore
	}
	if l.ForceDowntime {
		return ScalingDown
	}
	if l.ForceUptime {
		return ScalingUp
	}

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
			return ScalingDown
		}
		return ScalingIgnore
	}

	return ScalingNone
}

type Layers []Layer

// GetCurrentScaling gets the current scaling of the first layer that implements scaling
func (l Layers) GetCurrentScaling(maxGracePeriod time.Duration) (scaling, error) {
	for _, layer := range l {
		err := layer.checkForIncompatibleFields(maxGracePeriod)
		if err != nil {
			return ScalingIncompatible, fmt.Errorf("error found incompatible fields: %w", err)
		}

		layerScaling := layer.getCurrentScaling()
		if layerScaling == ScalingNone {
			continue
		}

		return layerScaling, nil
	}
	return ScalingNone, errNoScalingProvided
}
