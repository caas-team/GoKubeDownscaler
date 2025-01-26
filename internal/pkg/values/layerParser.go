package values

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
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

	envUpscalePeriod      = "UPSCALE_PERIOD"
	envUptime             = "DEFAULT_UPTIME"
	envDownscalePeriod    = "DOWNSCALE_PERIOD"
	envDowntime           = "DEFAULT_DOWNTIME"
	envExcludeNamespaces  = "EXCLUDE_NAMESPACES"
	envExcludeDeployments = "EXCLUDE_DEPLOYMENTS"
)

// ParseLayerFlags sets all flags corresponding to layer values to fill into l.
func (l *Layer) ParseLayerFlags() {
	flag.Var(
		&l.DownscalePeriod,
		"downscale-period",
		"period to scale down in (default: never, incompatible: UpscaleTime, DownscaleTime)",
	)
	flag.Var(
		&l.DownTime,
		"default-downtime",
		`timespans where workloads will be scaled down.
		outside of them they will be scaled up.
		(default: never, incompatible: UpscalePeriod, DownscalePeriod)`,
	)
	flag.Var(
		&l.UpscalePeriod,
		"upscale-period",
		"periods to scale up in (default: never, incompatible: UpscaleTime, DownscaleTime)",
	)
	flag.Var(
		&l.UpTime,
		"default-uptime",
		`timespans where workloads will be scaled up.
		outside of them they will be scaled down.
		(default: never, incompatible: UpscalePeriod, DownscalePeriod)`,
	)
	flag.Var(
		&l.Exclude,
		"explicit-include",
		"sets exclude on cli layer to true, makes it so namespaces or deployments have to specify downscaler/exclude=false (default: false)",
	)
	flag.Var(
		(*util.Int32Value)(&l.DownscaleReplicas),
		"downtime-replicas",
		"the replicas to scale down to (default: 0)",
	)
	flag.Var(
		(*util.DurationValue)(&l.GracePeriod),
		"grace-period",
		"the grace period between creation of workload until first downscale (default: 15min)",
	)

	if flag.Lookup("exclude-namespaces") == nil {
		flag.Var(
			&l.ExcludeNamespaces,
			"exclude-namespaces",
			"exclude namespaces from being scaled (default: kube-system,kube-downscaler)",
		)
	}

	if flag.Lookup("exclude-deployments") == nil {
		flag.Var(
			&l.ExcludeDeployments,
			"exclude-deployments",
			"exclude deployments from being scaled (optional)",
		)
	}
}

// GetLayerFromEnv fills l with all values from environment variables and checks for compatibility.
func (l *Layer) GetLayerFromEnv() error {
	err := util.GetEnvValue(envUpscalePeriod, &l.UpscalePeriod)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envUpscalePeriod, err)
	}

	err = util.GetEnvValue(envUptime, &l.UpTime)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envUptime, err)
	}

	err = util.GetEnvValue(envDownscalePeriod, &l.DownscalePeriod)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envDownscalePeriod, err)
	}

	err = util.GetEnvValue(envDowntime, &l.DownTime)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envDowntime, err)
	}

	err = util.GetEnvValue(envExcludeNamespaces, &l.ExcludeNamespaces)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envExcludeNamespaces, err)
	}

	err = util.GetEnvValue(envExcludeDeployments, &l.ExcludeDeployments)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envExcludeDeployments, err)
	}

	if err = l.CheckForIncompatibleFields(); err != nil {
		return fmt.Errorf("error: found incompatible fields: %w", err)
	}

	return nil
}

// GetLayerFromAnnotations fills l with all values from the annotations and checks for compatibility.
func (l *Layer) GetLayerFromAnnotations( //nolint: funlen,gocognit,gocyclo,cyclop // it is a big function and we can refactor it a bit but it should be fine for now
	annotations map[string]string,
	logEvent util.ResourceLogger,
	ctx context.Context,
) error {
	var err error

	if downscalePeriod, ok := annotations[annotationDownscalePeriod]; ok {
		err = l.DownscalePeriod.Set(downscalePeriod)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationDownscalePeriod, err)
			logEvent.ErrorInvalidAnnotation(annotationDownscalePeriod, err.Error(), ctx)

			return err
		}
	}

	if downtime, ok := annotations[annotationDowntime]; ok {
		err = l.DownTime.Set(downtime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationDowntime, err)
			logEvent.ErrorInvalidAnnotation(annotationDowntime, err.Error(), ctx)

			return err
		}
	}

	if upscalePeriod, ok := annotations[annotationUpscalePeriod]; ok {
		err = l.UpscalePeriod.Set(upscalePeriod)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationUpscalePeriod, err)
			logEvent.ErrorInvalidAnnotation(annotationUpscalePeriod, err.Error(), ctx)

			return fmt.Errorf("failed to parse %q annotation: %w", annotationUpscalePeriod, err)
		}
	}

	if uptime, ok := annotations[annotationUptime]; ok {
		err = l.UpTime.Set(uptime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationUptime, err)
			logEvent.ErrorInvalidAnnotation(annotationUptime, err.Error(), ctx)

			return err
		}
	}

	if exclude, ok := annotations[annotationExclude]; ok {
		err = l.Exclude.Set(exclude)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationExclude, err)
			logEvent.ErrorInvalidAnnotation(annotationExclude, err.Error(), ctx)

			return err
		}
	}

	if excludeUntil, ok := annotations[annotationExcludeUntil]; ok {
		l.ExcludeUntil, err = time.Parse(time.RFC3339, excludeUntil)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationExcludeUntil, err)
			logEvent.ErrorInvalidAnnotation(annotationExcludeUntil, err.Error(), ctx)

			return err
		}
	}

	if forceUptime, ok := annotations[annotationForceUptime]; ok {
		err = l.ForceUptime.Set(forceUptime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationForceUptime, err)
			logEvent.ErrorInvalidAnnotation(annotationForceUptime, err.Error(), ctx)

			return err
		}
	}

	if forceDowntime, ok := annotations[annotationForceDowntime]; ok {
		err = l.ForceDowntime.Set(forceDowntime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationForceDowntime, err)
			logEvent.ErrorInvalidAnnotation(annotationForceDowntime, err.Error(), ctx)

			return err
		}
	}

	if downscaleReplicasString, ok := annotations[annotationDownscaleReplicas]; ok {
		var downscaleReplicas int64

		downscaleReplicas, err = strconv.ParseInt(downscaleReplicasString, 10, 32)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationDownscaleReplicas, err)
			logEvent.ErrorInvalidAnnotation(annotationDownscaleReplicas, err.Error(), ctx)

			return err
		}

		// #nosec G115 // downscaleReplicas gets parsed as a 32 bit integer, so any errors that could be thrown here are already handled above
		l.DownscaleReplicas = int32(downscaleReplicas)
	}

	if gracePeriod, ok := annotations[annotationGracePeriod]; ok {
		err = (*util.DurationValue)(&l.GracePeriod).Set(gracePeriod)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationGracePeriod, err)
			logEvent.ErrorInvalidAnnotation(annotationGracePeriod, err.Error(), ctx)

			return err
		}
	}

	if err = l.CheckForIncompatibleFields(); err != nil {
		err = fmt.Errorf("error: found incompatible fields: %w", err)
		logEvent.ErrorIncompatibleFields(err.Error(), ctx)

		return err
	}

	return nil
}
