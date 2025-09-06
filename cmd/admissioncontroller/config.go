package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// runtimeConfiguration represents the runtime configuration for the admission controller.
type runtimeConfiguration struct {
	util.CommonRuntimeConfiguration
	// WebhookServiceName is the name of the Kubernetes service for the webhook.
	WebhookServiceName string
	// ClusterDomain is the cluster domain of the Kubernetes cluster.
	ClusterDomain string
	// CertSecretName is the name of the secret containing the TLS certs for the webhook.
	CertSecretName string
	// EnableCertRotation sets if cert rotation should be enabled.
	EnableCertRotation bool
	// Kubeconfig sets an optional kubeconfig to use for testing purposes instead of the in-cluster config.
	Kubeconfig string
}

func getDefaultConfig() *runtimeConfiguration {
	return &runtimeConfiguration{
		CommonRuntimeConfiguration: *util.GetDefaultConfig(),
		Kubeconfig:                 "",
	}
}

func (c *runtimeConfiguration) parseConfigFlags() {
	c.ParseCommonFlags()
	flag.BoolVar(
		&c.EnableCertRotation,
		"internal-cert-rotation",
		false,
		"enables internal cert rotation (default: false)",
	)
	flag.StringVar(
		&c.WebhookServiceName,
		"service-name",
		"",
		"service name to use for the webhook (default: go-kube-downscaler-webhook)",
	)
	flag.StringVar(
		&c.ClusterDomain,
		"cluster-domain",
		"cluster.local",
		"cluster domain of the Kubernetes cluster (default: cluster.local)",
	)
	flag.StringVar(
		&c.CertSecretName,
		"tls-secret-name",
		"",
		"secret name containing the TLS certs for the webhook (default: go-kube-downscaler-webhook)",
	)
	flag.StringVar(
		&c.Kubeconfig,
		"k",
		"",
		"kubeconfig to use instead of the in-cluster config (optional)",
	)
}

//nolint:nonamedreturns //required for function clarity
func initComponent() (runtimeConfig *runtimeConfiguration, scopeDefault, scopeCli, scopeEnv *values.Scope) {
	runtimeConfig = getDefaultConfig()
	runtimeConfig.parseConfigFlags()

	err := runtimeConfig.ParseConfigEnvVars()
	if err != nil {
		slog.Error("failed to parse env vars for runtimeConfig", "error", err)
		os.Exit(1)
	}

	scopeDefault, scopeCli, scopeEnv = values.InitScopes()

	if runtimeConfig.Debug || runtimeConfig.DryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err = scopeCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}

	slog.Debug("finished getting startup runtimeConfig",
		"envScope", scopeEnv,
		"cliScope", scopeCli,
		"runtimeConfig", runtimeConfig,
	)

	return runtimeConfig, scopeDefault, scopeCli, scopeEnv
}

func setupControllerRuntimeLogEncoding() zap.Options {
	return zap.Options{
		Development: true,
		EncoderConfigOptions: []zap.EncoderConfigOption{
			func(ec *zapcore.EncoderConfig) {
				ec.NameKey = ""   // remove logger name
				ec.CallerKey = "" // remove file info
				ec.EncodeLevel = zapcore.CapitalLevelEncoder
				ec.EncodeTime = zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05")
				ec.ConsoleSeparator = " "
			},
		},
	}
}

// setupConfig loads either in-cluster config or a kubeconfig if provided.
func setupConfig(kubeconfig string) *rest.Config {
	var cfg *rest.Config
	var err error

	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			slog.Error("failed to load kubeconfig", "path", kubeconfig, "error", err)
			os.Exit(1)
		}

		slog.Info("using provided kubeconfig", "path", kubeconfig)
	} else {
		cfg, err = config.GetConfig()
		if err != nil {
			slog.Error("failed to get Kubernetes config", "error", err)
			os.Exit(1)
		}

		slog.Info("using in-cluster or default kubeconfig")
	}

	return cfg
}
