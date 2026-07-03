package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes/admission"
	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type serverConfig struct {
	client               kubernetes.Client
	clientNoDryRun       kubernetes.Client
	scopeCli             *values.Scope
	scopeEnv             *values.Scope
	scopeDefault         *values.Scope
	config               *runtimeConfiguration
	includedResourcesSet map[string]struct{}
	admissionMetrics     *metrics.AdmissionMetrics
}

const (
	certDir                  = "/etc/webhook/tls"
	mutatingWebhookName      = "webhook.kube-downscaler.k8s"
	defaultCAName            = "KUBEDOWNSCALER"
	defaultCAOrg             = "KUBEDOWNSCALERORG"
	probeAddress             = ":8080"
	healthCheckName          = "healthz"
	readyCheckName           = "readyz"
	NamespaceCleanupInterval = 20 * time.Minute
)

func main() {
	config, scopeDefault, scopeCli, scopeEnv := initComponent()

	scheme := apimachineryruntime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	client, err := kubernetes.NewClient(config.Kubeconfig, config.DryRun, config.Qps, config.Burst)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	// Create a second client that is not in dry-run mode, for cert rotation which should always be performed
	// even when other operations are in dry-run mode
	clientNoDryRun, err := kubernetes.NewClient(config.Kubeconfig, false, config.Qps, config.Burst)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	admissionMetrics, bindAddress := initAdmissionMetrics(config)

	includedResourcesSet := toSet(config.IncludeResources)

	serverConfig := &serverConfig{
		client:               client,
		clientNoDryRun:       clientNoDryRun,
		scopeCli:             scopeCli,
		scopeEnv:             scopeEnv,
		scopeDefault:         scopeDefault,
		config:               config,
		includedResourcesSet: includedResourcesSet,
		admissionMetrics:     admissionMetrics,
	}

	opts := setupControllerRuntimeLogEncoding(config)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	baseCtx := ctrl.SetupSignalHandler()
	ctx, cancel := context.WithCancel(baseCtx)
	cfg := setupConfig(config.Kubeconfig)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		Metrics: server.Options{
			BindAddress: bindAddress,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    443,
			CertDir: certDir,
			TLSOpts: []func(tlsConfig *tls.Config){
				func(tlsConfig *tls.Config) {
					tlsConfig.MinVersion = tls.VersionTLS12
				},
			},
		}),
		HealthProbeBindAddress: probeAddress,
	})
	if err != nil {
		slog.Error("failed to create controller runtime", "error", err)
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck(healthCheckName, healthz.Ping); err != nil {
		slog.Error("failed to set up health check", "error", err)
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck(readyCheckName, healthz.Ping); err != nil {
		slog.Error("failed to set up ready check", "error", err)
		os.Exit(1)
	}

	certReady := make(chan struct{})

	if config.EnableCertRotation {
		slog.Info("setting up internal certificates rotation")

		certManager := kubernetes.CertManager{
			SecretName:          config.CertSecretName,
			CertDir:             certDir,
			WebhookService:      config.WebhookServiceName,
			K8sClusterDomain:    config.ClusterDomain,
			CAName:              defaultCAName,
			CAOrganization:      defaultCAOrg,
			MutatingWebhookName: mutatingWebhookName,
			Ready:               certReady,
			Client:              clientNoDryRun,
		}
		if err = certManager.AddCertificateRotation(ctx, mgr); err != nil {
			slog.Error("failed to add certificate rotation", "error", err)
			os.Exit(1)
		}

		go startManager(ctx, mgr)

		slog.Info("waiting for TLS certs to be ready")
		<-certManager.Ready
		slog.Info("TLS certs are ready")
	} else {
		slog.Warn("internal certificates rotation is not enabled, make sure certificate rotation is handled externally")
		close(certReady)

		go startManager(ctx, mgr)
	}

	slog.Info("serving webhook server")

	hookServer := mgr.GetWebhookServer()
	hookServer.Register("/validate-workloads", http.HandlerFunc(serverConfig.serveValidateWorkloads))
	startNamespaceCleanup(ctx, serverConfig, client, cancel, config.MetricsEnabled)

	<-ctx.Done()
}

// serveValidateWorkloads validates an admission request.
func (s *serverConfig) serveValidateWorkloads(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	slog.Debug("received validation request from uri", "requestURI", request.RequestURI)

	admissionHandler := admission.NewWorkloadMutationHandler(
		s.client,
		s.scopeCli,
		s.scopeEnv,
		s.scopeDefault,
		s.config.DryRun,
		&s.config.IncludeNamespaces,
		&s.config.IncludeLabels,
		&s.config.ExcludeNamespaces,
		&s.config.ExcludeWorkloads,
		s.includedResourcesSet,
		s.config.MetricsEnabled,
		s.admissionMetrics,
	)
	admissionHandler.HandleWorkloadMutation(ctx, writer, request)

	slog.Info("validation request was correctly processed")
}

func startManager(ctx context.Context, mgr manager.Manager) {
	if err := mgr.Start(ctx); err != nil {
		slog.Error("manager exited with error", "error", err)
		os.Exit(1)
	}
}

//nolint:nonamedreturns //required for function clarity
func initAdmissionMetrics(config *runtimeConfiguration) (admissionMetrics *metrics.AdmissionMetrics, bindingPort string) {
	if !config.MetricsEnabled {
		return nil, "0"
	}

	m := metrics.NewAdmissionMetrics(config.DryRun)
	m.RegisterAll()
	slog.Info("metrics configuration gathered")

	return m, ":8085"
}

func startNamespaceCleanup(
	ctx context.Context,
	cfg *serverConfig,
	client kubernetes.Client,
	cancel context.CancelFunc,
	metricsEnabled bool,
) {
	if metricsEnabled {
		slog.Debug("starting routine for deleted namespace metrics cleanup")

		errCh := make(chan error)

		go cleanDeletedNamespaceMetrics(ctx, cfg, client, errCh)

		go func() {
			for err := range errCh {
				slog.Error("webhook controller was not able to clean metrics for unused namespace", "error", err)
				cancel()

				return
			}
		}()
	}
}

func cleanDeletedNamespaceMetrics(ctx context.Context, config *serverConfig, client kubernetes.Client, errCh chan<- error) {
	var previousNamespace, currentNamespace map[string]struct{}
	var err error

	previousNamespace, err = client.GetNamespacesAsSet()
	if err != nil {
		errCh <- fmt.Errorf("failed to get namespaces on startup: %w", err)
		return
	}

	ticker := time.NewTicker(NamespaceCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentNamespace, err = client.GetNamespacesAsSet()
			if err != nil {
				errCh <- fmt.Errorf("failed to get namespaces during cleanup: %w", err)
				continue
			}

			for namespace := range previousNamespace {
				if _, exists := currentNamespace[namespace]; !exists {
					config.admissionMetrics.DeleteNamespaceMetrics(namespace)
				}
			}

			previousNamespace = currentNamespace

		case <-ctx.Done():
			return
		}
	}
}

func toSet(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, item := range items {
		m[item] = struct{}{}
	}

	return m
}
