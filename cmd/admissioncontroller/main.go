package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes/admission"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

type serverConfig struct {
	client       kubernetes.Client
	scopeCli     *values.Scope
	scopeEnv     *values.Scope
	scopeDefault *values.Scope
	config       *runtimeConfiguration
}

func main() {
	config, scopeDefault, scopeCli, scopeEnv := initComponent()

	client, err := kubernetes.NewClient("", config.DryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	serverConfig := &serverConfig{
		client:       client,
		scopeCli:     scopeCli,
		scopeEnv:     scopeEnv,
		scopeDefault: scopeDefault,
		config:       config,
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

		slog.Info("Listening on port 8080...")

		if err = httpServer.ListenAndServe(); err != nil {
			slog.Error("failed to start HTTP server", "error", err)
		}
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

	slog.Info("Listening on port 443...")

	err = httpsServer.ListenAndServeTLS(cert, key)
	if err != nil {
		slog.Error("failed to start HTTPS server", "error", err)
	}
}

// ServeHealth returns 200 when things are good.
func (s *serverConfig) serveHealth(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprint(w, "OK")
	if err != nil {
		return
	}
}

// ServeValidateWorkloads validates an admission request.
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
