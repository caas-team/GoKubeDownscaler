package values

import (
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

// GetLayerFromAnnotations makes a layer and fills it with all values from the annotations
func GetLayerFromAnnotations(annotations map[string]string) (Layer, error) {
	result := NewLayer()
	var err error

	if downscalePeriod, ok := annotations[annotationDownscalePeriod]; ok {
		err = result.DownscalePeriod.Set(downscalePeriod)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationDownscalePeriod, err)
		}
	}
	if downtime, ok := annotations[annotationDowntime]; ok {
		err = result.DownTime.Set(downtime)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationDowntime, err)
		}
	}
	if upscalePeriod, ok := annotations[annotationUpscalePeriod]; ok {
		err = result.UpscalePeriod.Set(upscalePeriod)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationUpscalePeriod, err)
		}
	}
	if uptime, ok := annotations[annotationUptime]; ok {
		err = result.UpTime.Set(uptime)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationUptime, err)
		}
	}
	if exclude, ok := annotations[annotationExclude]; ok {
		err = result.Exclude.Set(exclude)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationExclude, err)
		}
	}
	if excludeUntil, ok := annotations[annotationExcludeUntil]; ok {
		result.ExcludeUntil, err = time.Parse(time.RFC3339, excludeUntil)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationExcludeUntil, err)
		}
	}
	if forceUptime, ok := annotations[annotationForceUptime]; ok {
		result.ForceUptime, err = strconv.ParseBool(forceUptime)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationForceUptime, err)
		}
	}
	if forceDowntime, ok := annotations[annotationForceDowntime]; ok {
		result.ForceDowntime, err = strconv.ParseBool(forceDowntime)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationForceDowntime, err)
		}
	}
	if downscaleReplicas, ok := annotations[annotationDownscaleReplicas]; ok {
		result.DownscaleReplicas, err = strconv.Atoi(downscaleReplicas)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationDownscaleReplicas, err)
		}
	}
	if gracePeriod, ok := annotations[annotationGracePeriod]; ok {
		err = result.GracePeriod.Set(gracePeriod)
		if err != nil {
			return result, fmt.Errorf("failed to parse %q annotation: %w", annotationGracePeriod, err)
		}
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
	return result, nil
}
