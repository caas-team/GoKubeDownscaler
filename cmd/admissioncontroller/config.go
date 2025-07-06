package main

import (
	"log/slog"
	"os"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

// runtimeConfiguration represents the runtime configuration for the admission controller.
type runtimeConfiguration struct {
	util.CommonRuntimeConfiguration
}

func getDefaultConfig() *runtimeConfiguration {
	return &runtimeConfiguration{
		CommonRuntimeConfiguration: *util.GetDefaultConfig(),
	}
}

func (c *runtimeConfiguration) parseConfigFlags() {
	c.ParseCommonFlags()
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
