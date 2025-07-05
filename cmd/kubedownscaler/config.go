package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

// runtimeConfiguration represents the runtime configuration for the downscaler.
type runtimeConfiguration struct {
	util.CommonRuntimeConfiguration
	// Once sets if the scan should only run once.
	Once bool
	// LeaderElection sets if leader election should be performed.
	LeaderElection bool
	// Interval sets how long to wait between scans.
	Interval time.Duration
	// TimeAnnotation sets the annotation used for grace-period instead of creation time.
	TimeAnnotation string
	// MaxRetriesOnConflict sets the maximum number of retries on 409 errors.
	MaxRetriesOnConflict int
	// Kubeconfig sets an optional kubeconfig to use for testing purposes instead of the in-cluster config.
	Kubeconfig string
}

func getDefaultConfig() *runtimeConfiguration {
	return &runtimeConfiguration{
		CommonRuntimeConfiguration: *util.GetDefaultConfig(),
		Once:                       false,
		Interval:                   30 * time.Second,
		TimeAnnotation:             "",
		Kubeconfig:                 "",
	}
}

// ParseConfigFlags sets all cli flags required for the runtime configuration.
func (c *runtimeConfiguration) parseConfigFlags() {
	c.ParseCommonFlags()
	flag.BoolVar(
		&c.Once,
		"once",
		false,
		"run scan only once (default: false)",
	)
	flag.BoolVar(
		&c.LeaderElection,
		"leader-election",
		false,
		"enables leader election (default: false)",
	)
	flag.Var(
		(*util.DurationValue)(&c.Interval),
		"interval",
		"time between scans (default: 30s)",
	)
	flag.StringVar(
		&c.Kubeconfig,
		"k",
		"",
		"kubeconfig to use instead of the in-cluster config (optional)",
	)
	flag.StringVar(
		&c.TimeAnnotation,
		"deployment-time-annotation",
		"",
		"the annotation to use instead of creation time for grace period (optional)",
	)
	flag.IntVar(
		&c.MaxRetriesOnConflict,
		"max-retries-on-conflict",
		0,
		"maximum number of retries on 409 conflict errors (default: 0)",
	)
}

//nolint:nonamedreturns //required for function clarity
func initComponent() (config *runtimeConfiguration, scopeDefault, scopeCli, scopeEnv *values.Scope) {
	config = getDefaultConfig()
	config.parseConfigFlags()

	err := config.ParseConfigEnvVars()
	if err != nil {
		slog.Error("failed to parse env vars for config", "error", err)
		os.Exit(1)
	}

	scopeDefault, scopeCli, scopeEnv = values.InitScopes()

	if config.Debug || config.DryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err = scopeCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}

	slog.Debug("finished getting startup config",
		"envScope", scopeEnv,
		"cliScope", scopeCli,
		"config", config,
	)

	return config, scopeDefault, scopeCli, scopeEnv
}
