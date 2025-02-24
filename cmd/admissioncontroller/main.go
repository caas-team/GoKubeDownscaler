package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	// value defaults.
	defaultGracePeriod       = 15 * time.Minute
	defaultDownscaleReplicas = 0
)

type ServerConfig struct {
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

	s := &ServerConfig{
		client:   client,
		layerCli: &layerCli,
		layerEnv: &layerEnv,
		config:   config,
		ctx:      ctx,
	}

	http.HandleFunc("/validate-workloads", s.ServeValidateWorkloads)
	http.HandleFunc("/health", ServeHealth)

	if os.Getenv("TLS") == "true" {
		cert := "/etc/admission-webhook/tls/tls.crt"
		key := "/etc/admission-webhook/tls/tls.key"
		slog.Info("Listening on port 443...")
		slog.Error("error", http.ListenAndServeTLS(":443", cert, key, nil))
	} else {
		slog.Info("Listening on port 8080...")
		slog.Error("error", http.ListenAndServe(":8080", nil))
	}
}

// ServeHealth returns 200 when things are good
func ServeHealth(w http.ResponseWriter, r *http.Request) {
	slog.Info("healthy")
	fmt.Fprint(w, "OK")
}

// ServeValidateWorkloads validates an admission request
func (s *ServerConfig) ServeValidateWorkloads(w http.ResponseWriter, r *http.Request) {
	slog.Info("received validation request from uri", "requestURI", r.RequestURI)

	in, err := parseAdmissionReviewFromRequest(*r)
	if err != nil {
		slog.Error("error encountered while parsing the request", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	workload, err := scalable.ParseWorkloadFromAdmissionReview(strings.ToLower(in.Request.Kind.Kind), in)
	if err != nil {
		slog.Error("error encountered while parsing the workload", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := s.validateWorkloadReview(workload, in)
	if err != nil {
		e := fmt.Sprintf("could not generate admission response: %s", err)
		slog.Error(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	jout, err := json.Marshal(out)
	if err != nil {
		e := fmt.Sprintf("could not parse admission response: %s", err)
		slog.Error(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}

	slog.Debug("sending response")
	slog.Debug("%s", jout)
}

// parseAdmissionReviewFromRequest extracts an AdmissionReview from a http.Request if possible
func parseAdmissionReviewFromRequest(r http.Request) (*admissionv1.AdmissionReview, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Content-Type: %q should be %q",
			r.Header.Get("Content-Type"), "application/json")
	}

	bodybuf := new(bytes.Buffer)
	bodybuf.ReadFrom(r.Body)
	body := bodybuf.Bytes()

	if len(body) == 0 {
		return nil, fmt.Errorf("admission request body is empty")
	}

	var a admissionv1.AdmissionReview

	if err := json.Unmarshal(body, &a); err != nil {
		return nil, fmt.Errorf("could not parse admission review request: %v", err)
	}

	if a.Request == nil {
		return nil, fmt.Errorf("admission review can't be used: Request field is nil")
	}

	return &a, nil
}

func (s *ServerConfig) validateWorkloadReview(
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
) (*admissionv1.AdmissionReview, error) {

	val, reason, err := s.validateWorkload(workload)
	if err != nil {
		e := fmt.Sprintf("could not validate pod: %s", err)
		return reviewResponse(review.Request.UID, false, http.StatusBadRequest, e), err
	}

	if !val {
		return reviewResponse(review.Request.UID, false, http.StatusForbidden, reason), nil
	}

	return reviewResponse(review.Request.UID, true, http.StatusAccepted, reason), nil
}

func (s *ServerConfig) validateWorkload(workload scalable.Workload) (bool, string, error) {
	resourceLogger := kubernetes.NewResourceLogger(s.client, workload)

	workloadArray := []scalable.Workload{workload}

	workloads := scalable.FilterExcluded(workloadArray, s.config.IncludeLabels, s.config.ExcludeNamespaces, s.config.ExcludeWorkloads)
	slog.Info("scanning over workloads matching filters", "amount", len(workloads))

	/*var waitGroup sync.WaitGroup
	var lock sync.Mutex
	isExcluded := false
	for _, w := range workloads {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if !(w.GroupVersionKind() == workload.GroupVersionKind() && w.GetName() == workload.GetName() && w.GetNamespace() == workload.GetNamespace()){
				lock.Lock()
				isExcluded = true
				lock.Unlock()
			}
		}()
	}
	waitGroup.Wait()

	if isExcluded {
		return false, "workload is excluded", nil
	}*/

	namespaceAnnotations, err := s.client.GetNamespaceAnnotations(workload.GetNamespace(), s.ctx)
	if err != nil {
		return false, "failed to get namespace annotations", fmt.Errorf("failed to get namespace annotations: %w", err)
	}

	slog.Debug(
		"parsing workload layer from annotations",
		"annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerWorkload := values.NewLayer()
	if err = layerWorkload.GetLayerFromAnnotations(workload.GetAnnotations(), resourceLogger, s.ctx); err != nil {
		return false, "failed to process annotation", fmt.Errorf("failed to parse workload layer from annotations: %w", err)
	}

	slog.Debug(
		"parsing namespace layer from annotations",
		"annotations", namespaceAnnotations,
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerNamespace := values.NewLayer()
	if err = layerNamespace.GetLayerFromAnnotations(namespaceAnnotations, resourceLogger, s.ctx); err != nil {
		return false, "failed to process namespace annotation", fmt.Errorf("failed to parse namespace layer from annotations: %w", err)
	}

	layers := values.Layers{&layerWorkload, &layerNamespace, s.layerCli, s.layerEnv}

	slog.Debug("finished parsing all layers", "layers", layers, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	if layers.GetExcluded() {
		slog.Debug("workload is excluded", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return false, "workload is excluded", nil
	}

	return true, "workload is admitted", nil
}

// reviewResponse returns an AdmissionReview with the specified UID, allowed, httpCode, and reason
func reviewResponse(uid types.UID, allowed bool, httpCode int32, reason string) *admissionv1.AdmissionReview {
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:     uid,
			Allowed: allowed,
			Result: &metav1.Status{
				Code:    httpCode,
				Message: reason,
			},
		},
	}
}
