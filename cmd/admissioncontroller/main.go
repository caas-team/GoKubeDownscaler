package main

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	_ "time/tzdata"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes/admission"
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
	client         kubernetes.Client
	clientNoDryRun kubernetes.Client
	scopeCli       *values.Scope
	scopeEnv       *values.Scope
	scopeDefault   *values.Scope
	config         *runtimeConfiguration
}

const (
	certDir             = "/etc/webhook/tls"
	mutatingWebhookName = "webhook.kube-downscaler.k8s"
	defaultCAName       = "KUBEDOWNSCALER"
	defaultCAOrg        = "KUBEDOWNSCALERORG"
)

func main() {
	config, scopeDefault, scopeCli, scopeEnv := initComponent()

	scheme := apimachineryruntime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	client, err := kubernetes.NewClient(config.Kubeconfig, config.DryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	// Create a second client that is not in dry-run mode, for cert rotation which should always be performed
	// even when other operations are in dry-run mode
	clientNoDryRun, err := kubernetes.NewClient(config.Kubeconfig, false)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	serverConfig := &serverConfig{
		client:         client,
		clientNoDryRun: clientNoDryRun,
		scopeCli:       scopeCli,
		scopeEnv:       scopeEnv,
		scopeDefault:   scopeDefault,
		config:         config,
	}

	opts := setupControllerRuntimeLogEncoding()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctx := ctrl.SetupSignalHandler()
	cfg := setupConfig(config.Kubeconfig)

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
		Metrics: server.Options{
			BindAddress: "0", // metrics disabled
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
		HealthProbeBindAddress: ":8080",
	})
	if err != nil {
		slog.Error("failed to create controller runtime", "error", err)
		os.Exit(1)
	}

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		slog.Error("failed to set up health check", "error", err)
		os.Exit(1)
	}

	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
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

	<-ctx.Done()
}

// serveValidateWorkloads validates an admission request.
func (s *serverConfig) serveValidateWorkloads(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	slog.Debug("received validation request from uri", "requestURI", request.RequestURI)

	admissionHandler := admission.NewMutationHandler(
		s.client,
		s.scopeCli,
		s.scopeEnv,
		s.scopeDefault,
		&s.config.IncludeLabels,
		&s.config.ExcludeNamespaces,
		&s.config.ExcludeWorkloads,
	)
	admissionHandler.HandleMutation(ctx, writer, request)

	slog.Info("validation request was correctly processed")
}

func startManager(ctx context.Context, mgr manager.Manager) {
	if err := mgr.Start(ctx); err != nil {
		slog.Error("manager exited with error", "error", err)
		os.Exit(1)
	}
}
