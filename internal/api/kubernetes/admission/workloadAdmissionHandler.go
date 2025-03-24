package admission

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	admissionv1 "k8s.io/api/admission/v1"
)

// WorkloadAdmissionHandler is a struct that implements the admissionHandler interface.
type WorkloadAdmissionHandler struct {
	admissionHandler
	client       kubernetes.Client
	scopeCli     *values.Scope
	scopeEnv     *values.Scope
	scopeDefault *values.Scope
	config       *util.AdmissionControllerRuntimeConfiguration
	ctx          context.Context
}

// NewWorkloadAdmissionHandler creates a new WorkloadAdmissionHandler.
func NewWorkloadAdmissionHandler(
	client kubernetes.Client,
	scopeCli, scopeEnv, scopeDefault *values.Scope,
	config *util.AdmissionControllerRuntimeConfiguration,
	ctx context.Context,
) *WorkloadAdmissionHandler {
	return &WorkloadAdmissionHandler{
		client:       client,
		scopeCli:     scopeCli,
		scopeEnv:     scopeEnv,
		scopeDefault: scopeDefault,
		config:       config,
		ctx:          ctx,
	}
}

// HandleValidation handles the validation of a workload.
func (v *WorkloadAdmissionHandler) HandleValidation(writer http.ResponseWriter, request *http.Request) {
	input, err := parseAdmissionReviewFromRequest(request)
	if err != nil {
		slog.Error("error encountered while parsing the request", "error", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)

		return
	}

	workload, err := scalable.ParseWorkloadFromAdmissionReview(strings.ToLower(input.Request.Kind.Kind), input)
	if err != nil {
		slog.Error("error encountered while parsing the workload", "error", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)

		return
	}

	out, err := v.validateWorkload(workload, input)
	if err != nil {
		slog.Error("error encountered while validating workload", "error", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)

		return
	}

	sendAdmissionReviewResponse(writer, out)
}

// validateWorkload validates the workload and returns an AdmissionReview.
func (v *WorkloadAdmissionHandler) validateWorkload(
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
) (*admissionv1.AdmissionReview, error) {
	resourceLogger := kubernetes.NewResourceLogger(v.client, workload)

	slog.Debug("validating workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	// check if workload is excluded
	workloadArray := []scalable.Workload{workload}

	workloads := scalable.FilterExcluded(workloadArray, v.config.IncludeLabels, v.config.ExcludeNamespaces, v.config.ExcludeWorkloads)

	if len(workloads) == 0 {
		slog.Debug("workload is excluded from downscaling", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, allowing it"), nil
	}

	workload = workloads[0]

	slog.Info("scanning over workloads matching filters", "amount", len(workloads))
	slog.Debug("scanning over workloads matching filters", "amount", len(workloads))

	namespaceAnnotations, err := v.client.GetNamespaceAnnotations(workload.GetNamespace(), v.ctx)
	if err != nil {
		reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get namespace annotations")
	}

	slog.Debug(
		"parsing workload scope from annotations",
		"workload annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeWorkload := values.NewScope()
	if err := scopeWorkload.GetScopeFromAnnotations(workload.GetAnnotations(), resourceLogger, v.ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get scope from annotations"), err
	}

	slog.Debug(
		"parsing namespace scope from annotations",
		"namespace annotations", namespaceAnnotations,
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeNamespace := values.NewScope()
	if err := scopeNamespace.GetScopeFromAnnotations(namespaceAnnotations, resourceLogger, v.ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get scope from annotations"), err
	}

	scopes := values.Scopes{&scopeWorkload, &scopeNamespace, v.scopeCli, v.scopeEnv, v.scopeDefault}

	slog.Debug("finished parsing all scopes", "scopes", scopes, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	if scopes.GetExcluded() {
		slog.Debug("workload is excluded", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, allowing it"), nil
	}

	return reviewResponse(review.Request.UID, false, http.StatusBadRequest, "workload is not allowed during downscale time"), nil
}
