package values

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	annotationDownscalePeriod   = "downscaler/downscale-period"
	annotationDowntime          = "downscaler/downtime"
	annotationUpscalePeriod     = "downscaler/upscale-period"
	annotationUptime            = "downscaler/uptime"
	annotationExclude           = "downscaler/exclude"
	annotationExcludeUntil      = "downscaler/exclude-until"
	annotationForceUptime       = "downscaler/force-uptime"
	annotationForceDowntime     = "downscaler/force-downtime"
	annotationDownscaleReplicas = "downscaler/downscale-replicas"
	annotationGracePeriod       = "downscaler/grace-period"

	envUpscalePeriod   = "UPSCALE_PERIOD"
	envUptime          = "DEFAULT_UPTIME"
	envDownscalePeriod = "DOWNSCALE_PERIOD"
	envDowntime        = "DEFAULT_DOWNTIME"
)

type resourceLogger interface {
	// ErrorInvalidAnnotation adds an invalid annotation error on a resource
	ErrorInvalidAnnotation(id string, message string, ctx context.Context)
	// ErrorIncompatibleFields adds an incompatible fields error on a resource
	ErrorIncompatibleFields(message string, ctx context.Context)
}

// GetLayerFromAnnotations makes a layer and fills it with all values from the annotations
func GetLayerFromAnnotations(annotations map[string]string, logEvent resourceLogger, ctx context.Context) (Layer, error) {
	result := NewLayer()
	var err error

	if downscalePeriod, ok := annotations[annotationDownscalePeriod]; ok {
		err = result.DownscalePeriod.Set(downscalePeriod)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationDownscalePeriod, fmt.Sprintf("failed to parse %q annotation: %s", annotationDownscalePeriod, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationDownscalePeriod, err)
		}
	}
	if downtime, ok := annotations[annotationDowntime]; ok {
		err = result.DownTime.Set(downtime)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationDowntime, fmt.Sprintf("failed to parse %q annotation: %s", annotationDowntime, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationDowntime, err)
		}
	}
	if upscalePeriod, ok := annotations[annotationUpscalePeriod]; ok {
		err = result.UpscalePeriod.Set(upscalePeriod)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationUpscalePeriod, fmt.Sprintf("failed to parse %q annotation: %s", annotationUpscalePeriod, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationUpscalePeriod, err)
		}
	}
	if uptime, ok := annotations[annotationUptime]; ok {
		err = result.UpTime.Set(uptime)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationUptime, fmt.Sprintf("failed to parse %q annotation: %s", annotationUptime, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationUptime, err)
		}
	}
	if exclude, ok := annotations[annotationExclude]; ok {
		err = result.Exclude.Set(exclude)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationExclude, fmt.Sprintf("failed to parse %q annotation: %s", annotationExclude, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationExclude, err)
		}
	}
	if excludeUntil, ok := annotations[annotationExcludeUntil]; ok {
		result.ExcludeUntil, err = time.Parse(time.RFC3339, excludeUntil)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationExcludeUntil, fmt.Sprintf("failed to parse %q annotation: %s", annotationExcludeUntil, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationExcludeUntil, err)
		}
	}
	if forceUptime, ok := annotations[annotationForceUptime]; ok {
		err = result.ForceUptime.Set(forceUptime)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationForceUptime, fmt.Sprintf("failed to parse %q annotation: %s", annotationForceUptime, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationForceUptime, err)
		}
	}
	if forceDowntime, ok := annotations[annotationForceDowntime]; ok {
		err = result.ForceDowntime.Set(forceDowntime)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationForceDowntime, fmt.Sprintf("failed to parse %q annotation: %s", annotationForceDowntime, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationForceDowntime, err)
		}
	}
	if downscaleReplicasString, ok := annotations[annotationDownscaleReplicas]; ok {
		downscaleReplicas, err := strconv.ParseInt(downscaleReplicasString, 10, 32)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationDownscaleReplicas, fmt.Sprintf("failed to parse %q annotation: %s", annotationDownscaleReplicas, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationDownscaleReplicas, err)
		}
		// #nosec G115
		result.DownscaleReplicas = int32(downscaleReplicas)
	}
	if gracePeriod, ok := annotations[annotationGracePeriod]; ok {
		err = (*DurationValue)(&result.GracePeriod).Set(gracePeriod)
		if err != nil {
			logEvent.ErrorInvalidAnnotation(annotationGracePeriod, fmt.Sprintf("failed to parse %q annotation: %s", annotationGracePeriod, err.Error()), ctx)
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationGracePeriod, err)
		}
	}

	if err = result.CheckForIncompatibleFields(); err != nil {
		logEvent.ErrorIncompatibleFields(fmt.Sprintf("found incompatible fields: %s", err.Error()), ctx)
		return result, fmt.Errorf("error: found incompatible fields: %w", err)
	}

	return result, nil
}

// GetEnvValue gets the env value and puts it in flag.Value
func GetEnvValue(key string, value flag.Value) error {
	if val, ok := os.LookupEnv(key); ok {
		err := value.Set(val)
		if err != nil {
			return fmt.Errorf("failed to set value: %w", err)
		}
	}
	return nil
}

// GetLayerFromEnv makes a layer and fills it with all values from environment variables
func GetLayerFromEnv() (Layer, error) {
	result := NewLayer()
	err := GetEnvValue(envUpscalePeriod, &result.UpscalePeriod)
	if err != nil {
		return result, fmt.Errorf("error while getting %q environment variable: %w", envUpscalePeriod, err)
	}
	err = GetEnvValue(envUptime, &result.UpTime)
	if err != nil {
		return result, fmt.Errorf("error while getting %q environment variable: %w", envUptime, err)
	}
	err = GetEnvValue(envDownscalePeriod, &result.DownscalePeriod)
	if err != nil {
		return result, fmt.Errorf("error while getting %q environment variable: %w", envDownscalePeriod, err)
	}
	err = GetEnvValue(envDowntime, &result.DownTime)
	if err != nil {
		return result, fmt.Errorf("error while getting %q environment variable: %w", envDowntime, err)
	}

	if err = result.CheckForIncompatibleFields(); err != nil {
		return result, fmt.Errorf("error: found incompatible fields: %w", err)
	}

	return result, nil
}
