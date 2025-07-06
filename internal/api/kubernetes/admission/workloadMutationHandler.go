package admission

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	admissionv1 "k8s.io/api/admission/v1"
)

// MutationHandler is a struct that implements the admissionHandler interface.
type MutationHandler struct {
	client            kubernetes.Client
	scopeCli          *values.Scope
	scopeEnv          *values.Scope
	scopeDefault      *values.Scope
	includeLabels     *util.RegexList
	excludeNamespaces *util.RegexList
	excludeWorkloads  *util.RegexList
}

// NewMutationHandler creates a new MutationHandler.
func NewMutationHandler(
	client kubernetes.Client,
	scopeCli, scopeEnv, scopeDefault *values.Scope,
	includeLabels, excludeNamespaces, excludeWorkloads *util.RegexList,
) *MutationHandler {
	return &MutationHandler{
		client:            client,
		scopeCli:          scopeCli,
		scopeEnv:          scopeEnv,
		scopeDefault:      scopeDefault,
		includeLabels:     includeLabels,
		excludeNamespaces: excludeNamespaces,
		excludeWorkloads:  excludeWorkloads,
	}
}

// HandleMutation handles the validation of a workload.
func (v *MutationHandler) HandleMutation(ctx context.Context, writer http.ResponseWriter, request *http.Request) {
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

	slog.Info("received validation request for workload",
		"workload", workload.GetName(),
		"namespace", workload.GetNamespace(),
		"kind", workload.GroupVersionKind().Kind,
	)

	out, err := v.evaluateMutation(ctx, workload, input)
	if err != nil {
		slog.Error("error encountered while validating workload", "error", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)

		return
	}

	sendAdmissionReviewResponse(writer, out)
}

// evaluateMutation validates the workload and returns an AdmissionReview.
//
//nolint:govet //needed to reassign the error to a new variable
func (v *MutationHandler) evaluateMutation(
	ctx context.Context,
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
) (*admissionv1.AdmissionReview, error) {
	resourceLogger := kubernetes.NewResourceLoggerForWorkload(v.client, workload)

	slog.Info("evaluating mutation on workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	// check if workload is excluded
	workloadArray := []scalable.Workload{workload}

	workloads := scalable.FilterExcluded(workloadArray, *v.includeLabels, *v.excludeNamespaces, *v.excludeWorkloads)

	if len(workloads) == 0 {
		slog.Debug("workload is excluded from downscaling", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, doesn't need mutation"), nil
	}

	workload = workloads[0]

	slog.Debug("scanning over workloads matching filters", "amount", len(workloads))

	namespaceAnnotations, err := v.client.GetNamespaceAnnotations(workload.GetNamespace(), ctx)
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
	if err := scopeWorkload.GetScopeFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get scope from annotations"), err
	}

	slog.Debug(
		"parsing namespace scope from annotations",
		"namespace annotations", namespaceAnnotations,
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeNamespace := values.NewScope()
	if err := scopeNamespace.GetScopeFromAnnotations(namespaceAnnotations, resourceLogger, ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get scope from annotations"), err
	}

	scopes := values.Scopes{scopeWorkload, scopeNamespace, v.scopeCli, v.scopeEnv, v.scopeDefault}

	slog.Debug("finished parsing all scopes", "scopes", scopes, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	if scopes.GetExcluded() {
		slog.Info("workload is excluded from mutation", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, doesn't need mutation"), nil
	}

	slog.Info("workload matches mutation condition, scaling it down", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	response, err := mutateWorkload(workload, review, scopes)
	if err != nil {
		return response, err
	}

	return response, nil
}

// mutateWorkload mutates the workload by scaling it down based on the scopes.
func mutateWorkload(
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
	scopes values.Scopes,
) (*admissionv1.AdmissionReview, error) {
	// generate a deep copy of the workload to be able to generate a comparison patch
	workloadCopy, err := scalable.DeepCopyWorkload(workload)
	if err != nil {
		slog.Error("failed to deep copy workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to deep copy workload"), err
	}

	downscaleReplicas, err := scopes.GetDownscaleReplicas()
	if err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get downscale replicas"), err
	}

	err = workloadCopy.ScaleDown(downscaleReplicas)
	if err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to scale down workload"), err
	}

	patch, err := scalable.CompareWorkloads(workload, workloadCopy)
	if err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to compare workload"), err
	}

	slog.Debug("comparison patch correctly generated", "patch", patch.String())

	// convert the patch into JSON format
	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to marshal patch"), err
	}

	return patchReviewResponse(review.Request.UID, jsonPatch)
}
