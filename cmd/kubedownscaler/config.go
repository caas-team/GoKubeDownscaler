package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
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
	// MaxRetriesOnConflict sets the maximum number of retries on 409 errors.
	MaxRetriesOnConflict int
}

func getDefaultConfig() *runtimeConfiguration {
	return &runtimeConfiguration{
		CommonRuntimeConfiguration: *util.GetDefaultConfig(),
		Once:                       false,
		Interval:                   30 * time.Second,
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

	logLevel := slog.LevelInfo
	if config.Debug || config.DryRun {
		logLevel = slog.LevelDebug
	}

	if config.JsonLogs {
		opts := &slog.HandlerOptions{
			Level: logLevel,
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					a.Key = "severity"
				}

				return a
			},
		}

		logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
		slog.SetDefault(logger)
	}

	if config.Debug || config.DryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err = scopeCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}

	slog.Debug(
		"finished getting startup config",
		"envScope", scopeEnv,
		"cliScope", scopeCli,
		"config", config,
	)

	return config, scopeDefault, scopeCli, scopeEnv
}

// String gets the string representation of the runtime configuration.
func (c *runtimeConfiguration) String() string {
	if c == nil {
		return "<nil>"
	}

	var builder strings.Builder

	builder.WriteString("[")

	fmt.Fprintf(&builder, "dryRun:%t ", c.DryRun)
	fmt.Fprintf(&builder, "debug:%t ", c.Debug)
	fmt.Fprintf(&builder, "includeNamespaces:%v ", c.IncludeNamespaces)
	fmt.Fprintf(&builder, "includeResources:%v ", c.IncludeResources)
	fmt.Fprintf(&builder, "excludeNamespaces:%v ", c.ExcludeNamespaces)
	fmt.Fprintf(&builder, "excludeWorkloads:%v ", c.ExcludeWorkloads)
	fmt.Fprintf(&builder, "includeLabels:%v ", c.IncludeLabels)
	fmt.Fprintf(&builder, "timeAnnotation:%q ", c.TimeAnnotation)
	fmt.Fprintf(&builder, "metricsEnabled:%t ", c.MetricsEnabled)
	fmt.Fprintf(&builder, "jsonLogs:%t ", c.JsonLogs)
	fmt.Fprintf(&builder, "qps:%g ", c.Qps)
	fmt.Fprintf(&builder, "burst:%d ", c.Burst)
	fmt.Fprintf(&builder, "kubeconfig:%q ", c.Kubeconfig)
	fmt.Fprintf(&builder, "once:%t ", c.Once)
	fmt.Fprintf(&builder, "leaderElection:%t ", c.LeaderElection)
	fmt.Fprintf(&builder, "interval:%s ", c.Interval)
	fmt.Fprintf(&builder, "maxRetriesOnConflict:%d", c.MaxRetriesOnConflict)

	builder.WriteString("]")

	return builder.String()
}
