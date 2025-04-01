package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"
	_ "time/tzdata"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes/admission"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

const (
	// value defaults.
	defaultGracePeriod       = 15 * time.Minute
	defaultDownscaleReplicas = 0
)

type serverConfig struct {
	client       kubernetes.Client
	scopeCli     *values.Scope
	scopeEnv     *values.Scope
	scopeDefault *values.Scope
	config       *util.AdmissionControllerRuntimeConfiguration
	ctx          context.Context
}

func main() {
	ctx := context.Background()

	config := &util.AdmissionControllerRuntimeConfiguration{
		DryRun:            false,
		Debug:             false,
		IncludeNamespaces: nil,
		IncludeResources:  []string{"deployments"},
		ExcludeNamespaces: util.RegexList{regexp.MustCompile("kube-system"), regexp.MustCompile("kube-downscaler")},
		ExcludeWorkloads:  nil,
		IncludeLabels:     nil,
	}

	config.ParseConfigFlags()

	if err := config.ParseConfigEnvVars(); err != nil {
		slog.Error("failed to parse env vars for config", "error", err)
		os.Exit(1)
	}

	scopeDefault := values.GetDefaultScope()
	scopeCli := values.NewScope()
	scopeEnv := values.NewScope()

	if err := scopeEnv.GetScopeFromEnv(); err != nil {
		slog.Error("failed to get scope from env", "error", err)
		os.Exit(1)
	}

	// Set defaults for scopes
	scopeCli.GracePeriod = defaultGracePeriod
	scopeCli.DownscaleReplicas = defaultDownscaleReplicas
	scopeCli.ParseScopeFlags()

	flag.Parse()

	if config.Debug || config.DryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err := scopeCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewClient(config.Kubeconfig, config.DryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	serverConfig := &serverConfig{
		client:       client,
		scopeCli:     &scopeCli,
		scopeEnv:     &scopeEnv,
		scopeDefault: scopeDefault,
		config:       config,
		ctx:          ctx,
	}

	http.HandleFunc("/validate-workloads", serverConfig.serveValidateWorkloads)
	http.HandleFunc("/healthz", serverConfig.serveHealth)

	// Start the http server
	go func() {
		httpServer := &http.Server{
			Addr:         ":8080",
			Handler:      nil,              // You can set your handler here if needed
			ReadTimeout:  10 * time.Second, // Set read timeout
			WriteTimeout: 10 * time.Second, // Set write timeout
			IdleTimeout:  60 * time.Second, // Set idle timeout
		}

		if err = httpServer.ListenAndServe(); err != nil {
			slog.Error("failed to start HTTP server", "error", err)
		}

		slog.Info("Listening on port 8080...")
	}()

	// Start the https server
	cert := "/etc/admission-webhook/tls/tls.crt"
	key := "/etc/admission-webhook/tls/tls.key"

	httpsServer := &http.Server{
		Addr:         ":443",
		Handler:      nil,              // You can set your handler here if needed
		ReadTimeout:  10 * time.Second, // Set read timeout
		WriteTimeout: 10 * time.Second, // Set write timeout
		IdleTimeout:  60 * time.Second, // Set idle timeout
	}

	err = httpsServer.ListenAndServeTLS(cert, key)
	if err != nil {
		slog.Error("failed to start HTTPS server", "error", err)
	}

	slog.Info("Listening on port 443...")
}

// ServeHealth returns 200 when things are good.
func (s *serverConfig) serveHealth(w http.ResponseWriter, _ *http.Request) {
	slog.Debug("healthy")

	_, err := fmt.Fprint(w, "OK")
	if err != nil {
		return
	}
}

// ServeValidateWorkloads validates an admission request.
func (s *serverConfig) serveValidateWorkloads(writer http.ResponseWriter, request *http.Request) {
	slog.Info("received validation request from uri", "requestURI", request.RequestURI)

	a := admission.NewWorkloadAdmissionHandler(s.client, s.scopeCli, s.scopeEnv, s.scopeDefault, s.config, s.ctx)
	a.HandleValidation(writer, request)

	slog.Info("validation request was processed", "requestURI", request.RequestURI)
}
