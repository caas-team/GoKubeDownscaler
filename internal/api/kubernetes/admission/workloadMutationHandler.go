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

	slog.Debug(
		"parsing workload scope from annotations",
		"workload annotations", workload.GetAnnotations(),
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeWorkload := values.NewScope()
	if err := scopeWorkload.GetScopeFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx); err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get workload scope from annotations"), err
	}

	slog.Debug(
		"parsing namespace scope from workload",
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeNamespace, err := v.client.GetNamespaceScope(workload, ctx)
	if err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get namespace scope from annotations"), err
	}

	scopes := values.Scopes{scopeWorkload, scopeNamespace, v.scopeCli, v.scopeEnv, v.scopeDefault}

	slog.Debug("finished parsing all scopes", "scopes", scopes, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	if scopes.GetExcluded() {
		slog.Info("workload is excluded from mutation", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, doesn't need mutation"), nil
	}

	scaling := scopes.GetCurrentScaling()

	response, err := evaluateScalingConditions(scaling, workload, scopes, review)
	if err != nil {
		return response, err
	}

	return response, nil
}

// evaluateScalingConditions scales the given workload according to the given wanted scaling state.
func evaluateScalingConditions(
	scaling values.Scaling,
	workload scalable.Workload,
	scopes values.Scopes,
	review *admissionv1.AdmissionReview,
) (*admissionv1.AdmissionReview, error) {
	if scaling == values.ScalingNone {
		slog.Debug("scaling is not set by any scope, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "scaling configuration is not set for workload"), nil
	}

	if scaling == values.ScalingIgnore {
		slog.Debug("scaling is ignored, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "scaling configuration is ignored for workload"), nil
	}

	if scaling == values.ScalingMultiple {
		err := newScalingInvalidError(
			`scaling values matched to multiple states.
this is the result of a faulty configuration where on a scope there is multiple values with the same priority
setting different scaling states at the same time (e.g. downtime-period and uptime-period or force-downtime and force-uptime)`,
		)

		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "scaling configuration is invalid for workload"), err
	}

	if scaling == values.ScalingDown {
		slog.Info("mutating workload matching scaling down condition", "workload", workload.GetName(), "namespace", workload.GetNamespace())

		downscaleReplicas, err := scopes.GetDownscaleReplicas()
		if err != nil {
			return reviewResponse(review.Request.UID, true, http.StatusAccepted, "failed to get downscaleReplicas"), err
		}

		response, err := mutateWorkload(workload, review, downscaleReplicas)
		if err != nil {
			return response, err
		}

		return response, nil
	}

	if scaling == values.ScalingUp {
		slog.Debug("workload matches scaling up conditions, skipping", "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload matches scaling up conditions"), nil
	}

	return reviewResponse(review.Request.UID, true, http.StatusAccepted, "workload doesn't match any scaling condition"), nil
}

// mutateWorkload mutates the workload by scaling it down based on the scopes.
func mutateWorkload(
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
	downscaleReplicas values.Replicas,
) (*admissionv1.AdmissionReview, error) {
	// generate a deep copy of the workload to be able to generate a comparison patch
	workloadCopy, err := workload.Copy()
	if err != nil {
		slog.Error("failed to deep copy workload", "error", err, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to deep copy workload"), err
	}

	err = workloadCopy.ScaleDown(downscaleReplicas)
	if err != nil {
		return reviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to scale down workload"), err
	}

	patch, err := workload.Compare(workloadCopy)
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
