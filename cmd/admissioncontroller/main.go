package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes/admission"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"
)

const (
	// value defaults.
	defaultGracePeriod       = 15 * time.Minute
	defaultDownscaleReplicas = 0
)

type serverConfig struct {
	client   kubernetes.Client
	layerCli *values.Layer
	layerEnv *values.Layer
	config   *util.AdmissionControllerRuntimeConfiguration
	ctx      context.Context
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

	layerCli := values.NewLayer()
	layerEnv := values.NewLayer()

	if err := layerEnv.GetLayerFromEnv(); err != nil {
		slog.Error("failed to get layer from env", "error", err)
		os.Exit(1)
	}

	// Set defaults for layers
	layerCli.GracePeriod = defaultGracePeriod
	layerCli.DownscaleReplicas = defaultDownscaleReplicas
	layerCli.ParseLayerFlags()

	flag.Parse()

	if config.Debug || config.DryRun {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err := layerCli.CheckForIncompatibleFields(); err != nil {
		slog.Error("found incompatible fields", "error", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewClient(config.Kubeconfig, config.DryRun)
	if err != nil {
		slog.Error("failed to create new Kubernetes client", "error", err)
		os.Exit(1)
	}

	s := &serverConfig{
		client:   client,
		layerCli: &layerCli,
		layerEnv: &layerEnv,
		config:   config,
		ctx:      ctx,
	}

	http.HandleFunc("/validate-workloads", s.serveValidateWorkloads)
	http.HandleFunc("/health", s.serveHealth)

	// Start the server
	cert := "/etc/admission-webhook/tls/tls.crt"
	key := "/etc/admission-webhook/tls/tls.key"
	slog.Info("Listening on port 443...")
	slog.Error("error", http.ListenAndServeTLS(":443", cert, key, nil))

}

// ServeHealth returns 200 when things are good
func (s *serverConfig)  serveHealth(w http.ResponseWriter, r *http.Request) {
	slog.Info("healthy")
	fmt.Fprint(w, "OK")
}

// ServeValidateWorkloads validates an admission request
func (s *serverConfig) serveValidateWorkloads(w http.ResponseWriter, r *http.Request) {
	slog.Info("received validation request from uri", "requestURI", r.RequestURI)

	a := admission.NewWorkloadAdmissionHandler(s.client, s.layerCli, s.layerEnv, s.config, s.ctx)
	a.HandleValidation(w, r)

	slog.Error("validation request processed", "requestURI", r.RequestURI)
}
