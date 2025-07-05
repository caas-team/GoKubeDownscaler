package values

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
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
	annotationScaleChildren     = "downscaler/scale-children"

	envUpscalePeriod   = "UPSCALE_PERIOD"
	envUptime          = "DEFAULT_UPTIME"
	envDownscalePeriod = "DOWNSCALE_PERIOD"
	envDowntime        = "DEFAULT_DOWNTIME"
)

// ParseScopeFlags sets all flags corresponding to scope values to fill into l.
func (s *Scope) ParseScopeFlags() {
	flag.Var(
		&s.DownscalePeriod,
		"downscale-period",
		"period to scale down in (default: none, incompatible: UpscaleTime, DownscaleTime)",
	)
	flag.Var(
		&s.DownTime,
		"default-downtime",
		`timespans where workloads will be scaled down.
		outside of them they will be scaled up.
		(default: none, incompatible: UpscalePeriod, DownscalePeriod)`,
	)
	flag.Var(
		&s.ForceDowntime,
		"force-downtime",
		`timespans where workloads will be forced to be scaled down.
		(default: none)`,
	)
	flag.Var(
		&s.UpscalePeriod,
		"upscale-period",
		"periods to scale up in (default: none, incompatible: UpscaleTime, DownscaleTime)",
	)
	flag.Var(
		&s.UpTime,
		"default-uptime",
		`timespans where workloads will be scaled up.
		outside of them they will be scaled down.
		(default: none, incompatible: UpscalePeriod, DownscalePeriod)`,
	)
	flag.Var(
		&s.ForceUptime,
		"force-uptime",
		`timespans where workloads will be forced to be scaled up.
		(default: none)`,
	)
	flag.Var(
		&s.Exclude,
		"explicit-include",
		"sets exclude on cli scope to true, makes it so namespaces or deployments have to specify downscaler/exclude=false (default: false)",
	)
	flag.Var(
		&ReplicasValue{Replicas: &s.DownscaleReplicas},
		"downtime-replicas",
		"the replicas to scale down to (default: 0)",
	)
	flag.Var(
		(*util.DurationValue)(&s.GracePeriod),
		"grace-period",
		"the grace period between creation of workload until first downscale (default: 15min)",
	)
	flag.Var(
		&s.ScaleChildren,
		"scale-children",
		"if set to true, the ownerReference will immediately trigger scaling of children workloads when applicable (default: false)",
	)
}

// GetScopeFromEnv fills l with all values from environment variables and checks for compatibility.
func (s *Scope) GetScopeFromEnv() error {
	err := util.GetEnvValue(envUpscalePeriod, &s.UpscalePeriod)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envUpscalePeriod, err)
	}

	err = util.GetEnvValue(envUptime, &s.UpTime)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envUptime, err)
	}

	err = util.GetEnvValue(envDownscalePeriod, &s.DownscalePeriod)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envDownscalePeriod, err)
	}

	err = util.GetEnvValue(envDowntime, &s.DownTime)
	if err != nil {
		return fmt.Errorf("error while getting %q environment variable: %w", envDowntime, err)
	}

	if err = s.CheckForIncompatibleFields(); err != nil {
		return fmt.Errorf("error: found incompatible fields: %w", err)
	}

	return nil
}

// GetScopeFromAnnotations fills l with all values from the annotations and checks for compatibility.
func (s *Scope) GetScopeFromAnnotations( //nolint: funlen,gocognit,cyclop,gocyclo // it is a big function and we can refactor it a bit but it should be fine for now
	annotations map[string]string,
	logEvent util.ResourceLogger,
	ctx context.Context,
) error {
	var err error

	if downscalePeriod, ok := annotations[annotationDownscalePeriod]; ok {
		err = s.DownscalePeriod.Set(downscalePeriod)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationDownscalePeriod, err)
			logEvent.ErrorInvalidAnnotation(annotationDownscalePeriod, err.Error(), ctx)

			return err
		}
	}

	if downtime, ok := annotations[annotationDowntime]; ok {
		err = s.DownTime.Set(downtime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationDowntime, err)
			logEvent.ErrorInvalidAnnotation(annotationDowntime, err.Error(), ctx)

			return err
		}
	}

	if upscalePeriod, ok := annotations[annotationUpscalePeriod]; ok {
		err = s.UpscalePeriod.Set(upscalePeriod)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationUpscalePeriod, err)
			logEvent.ErrorInvalidAnnotation(annotationUpscalePeriod, err.Error(), ctx)

			return fmt.Errorf("failed to parse %q annotation: %w", annotationUpscalePeriod, err)
		}
	}

	if uptime, ok := annotations[annotationUptime]; ok {
		err = s.UpTime.Set(uptime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationUptime, err)
			logEvent.ErrorInvalidAnnotation(annotationUptime, err.Error(), ctx)

			return err
		}
	}

	if exclude, ok := annotations[annotationExclude]; ok {
		err = s.Exclude.Set(exclude)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationExclude, err)
			logEvent.ErrorInvalidAnnotation(annotationExclude, err.Error(), ctx)

			return err
		}
	}

	if excludeUntilString, ok := annotations[annotationExcludeUntil]; ok {
		var excludeUntil time.Time

		excludeUntil, err = time.Parse(time.RFC3339, excludeUntilString)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationExcludeUntil, err)
			logEvent.ErrorInvalidAnnotation(annotationExcludeUntil, err.Error(), ctx)

			return err
		}

		s.ExcludeUntil = &excludeUntil
	}

	if forceUptime, ok := annotations[annotationForceUptime]; ok {
		err = s.ForceUptime.Set(forceUptime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationForceUptime, err)
			logEvent.ErrorInvalidAnnotation(annotationForceUptime, err.Error(), ctx)

			return err
		}
	}

	if forceDowntime, ok := annotations[annotationForceDowntime]; ok {
		err = s.ForceDowntime.Set(forceDowntime)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationForceDowntime, err)
			logEvent.ErrorInvalidAnnotation(annotationForceDowntime, err.Error(), ctx)

			return err
		}
	}

	if downscaleReplicasString, ok := annotations[annotationDownscaleReplicas]; ok {
		replicasVal := ReplicasValue{Replicas: &s.DownscaleReplicas}

		if err = replicasVal.Set(downscaleReplicasString); err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationDownscaleReplicas, err)
			logEvent.ErrorInvalidAnnotation(annotationDownscaleReplicas, err.Error(), ctx)

			return err
		}

		s.DownscaleReplicas = *replicasVal.Replicas
	}

	if gracePeriod, ok := annotations[annotationGracePeriod]; ok {
		err = (*util.DurationValue)(&s.GracePeriod).Set(gracePeriod)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationGracePeriod, err)
			logEvent.ErrorInvalidAnnotation(annotationGracePeriod, err.Error(), ctx)

			return err
		}
	}

	if scaleChildrenString, ok := annotations[annotationScaleChildren]; ok {
		err = s.ScaleChildren.Set(scaleChildrenString)
		if err != nil {
			err = fmt.Errorf("failed to parse %q annotation: %w", annotationScaleChildren, err)
			logEvent.ErrorInvalidAnnotation(annotationScaleChildren, err.Error(), ctx)

			return err
		}
	}

	if err = s.CheckForIncompatibleFields(); err != nil {
		err = fmt.Errorf("error: found incompatible fields: %w", err)
		logEvent.ErrorIncompatibleFields(err.Error(), ctx)

		return err
	}

	return nil
}

//nolint:nonamedreturns //required for function clarity
func InitScopes() (scopeDefault, scopeCli, scopeEnv *Scope) {
	scopeDefault = GetDefaultScope()
	scopeCli = NewScope()
	scopeEnv = NewScope()

	err := scopeEnv.GetScopeFromEnv()
	if err != nil {
		slog.Error("failed to get scope from env", "error", err)
		os.Exit(1)
	}

	scopeCli.ParseScopeFlags()

	flag.Parse()

	return scopeDefault, scopeCli, scopeEnv
}
