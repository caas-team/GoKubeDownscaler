package admission

import (
	"context"
	"fmt"
	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	admissionv1 "k8s.io/api/admission/v1"
	"log/slog"
	"net/http"
	"strings"
)

// WorkloadAdmissionHandler is a struct that implements the admissionHandler interface
type WorkloadAdmissionHandler struct {
	admissionHandler
	client   kubernetes.Client
	layerCli *values.Layer
	layerEnv *values.Layer
	config   *util.AdmissionControllerRuntimeConfiguration
	ctx      context.Context
}

// NewWorkloadAdmissionHandler creates a new WorkloadAdmissionHandler
func NewWorkloadAdmissionHandler(client kubernetes.Client, layerCli *values.Layer, layerEnv *values.Layer, config *util.AdmissionControllerRuntimeConfiguration, ctx context.Context) *WorkloadAdmissionHandler {
	return &WorkloadAdmissionHandler{
		client:   client,
		layerCli: layerCli,
		layerEnv: layerEnv,
		config:   config,
		ctx:      ctx,
	}
}

// HandleValidation handles the validation of a workload
func (v WorkloadAdmissionHandler) HandleValidation(w http.ResponseWriter, r *http.Request) {

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

	out, err := v.validateWorkload(workload, in)
	if err != nil {
		e := fmt.Sprintf("could not generate admission response: %s", err)
		slog.Error(e)
		http.Error(w, e, http.StatusInternalServerError)
		return
	}

	sendAdmissionReviewResponse(w, err, out)

}

// validateWorkload validates the workload and returns an AdmissionReview
func (v WorkloadAdmissionHandler) validateWorkload(workload scalable.Workload, review *admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error) {
	resourceLogger := kubernetes.NewResourceLogger(v.client, workload)

	slog.Debug("validating workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	workloadArray := []scalable.Workload{workload}

	workloads := scalable.FilterExcluded(workloadArray, v.config.IncludeLabels, v.config.ExcludeNamespaces, v.config.ExcludeWorkloads)
	slog.Debug("scanning over workloads matching filters", "amount", len(workloads))

	namespaceAnnotations, err := v.client.GetNamespaceAnnotations(workload.GetNamespace(), v.ctx)
	if err != nil {
		reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get namespace annotations")
	}

	slog.Debug(
		"parsing workload layer from annotations",
		"workload annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerWorkload := values.NewLayer()
	if err = layerWorkload.GetLayerFromAnnotations(workload.GetAnnotations(), resourceLogger, v.ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get layer from annotations"), err
	}

	slog.Debug(
		"parsing namespace layer from annotations",
		"namespace annotations", namespaceAnnotations,
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	layerNamespace := values.NewLayer()
	if err = layerNamespace.GetLayerFromAnnotations(namespaceAnnotations, resourceLogger, v.ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get layer from annotations"), err
	}

	layers := values.Layers{&layerWorkload, &layerNamespace, v.layerCli, v.layerEnv}

	slog.Debug("finished parsing all layers", "layers", layers, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	if layers.GetExcluded() {
		slog.Debug("workload is excluded", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, allowing it"), nil
	}

	return reviewResponse(review.Request.UID, false, http.StatusBadRequest, "workload is not allowed during downscale time"), nil
}
