package admission

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	admissionv1 "k8s.io/api/admission/v1"
)

// WorkloadMutationHandler is a struct that implements the admissionHandler interface.
type WorkloadMutationHandler struct {
	client              kubernetes.Client
	scopeCli            *values.Scope
	scopeEnv            *values.Scope
	scopeDefault        *values.Scope
	includeNamespaces   *[]string
	dryRun              bool
	includeLabels       *util.RegexList
	excludeNamespaces   *util.RegexList
	excludeWorkloads    *util.RegexList
	includeResourcesSet map[string]struct{}
	metricsEnabled      bool
	admissionMetrics    *metrics.AdmissionMetrics
}

// NewWorkloadMutationHandler creates a new WorkloadMutationHandler.
func NewWorkloadMutationHandler(
	client kubernetes.Client,
	scopeCli, scopeEnv, scopeDefault *values.Scope,
	dryRun bool,
	includeNamespaces *[]string,
	includeLabels, excludeNamespaces, excludeWorkloads *util.RegexList,
	includeResources map[string]struct{},
	metricsEnabled bool,
	admissionMetrics *metrics.AdmissionMetrics,
) *WorkloadMutationHandler {
	return &WorkloadMutationHandler{
		client:              client,
		scopeCli:            scopeCli,
		scopeEnv:            scopeEnv,
		scopeDefault:        scopeDefault,
		dryRun:              dryRun,
		includeNamespaces:   includeNamespaces,
		includeLabels:       includeLabels,
		excludeNamespaces:   excludeNamespaces,
		excludeWorkloads:    excludeWorkloads,
		includeResourcesSet: includeResources,
		metricsEnabled:      metricsEnabled,
		admissionMetrics:    admissionMetrics,
	}
}

// HandleWorkloadMutation handles the validation of a workload.
func (v *WorkloadMutationHandler) HandleWorkloadMutation(ctx context.Context, writer http.ResponseWriter, request *http.Request) {
	input, err := parseAdmissionReviewFromRequest(request)
	if err != nil {
		slog.Error("error encountered while parsing the request", "error", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)

		return
	}

	workload, err := scalable.ParseWorkloadFromRawObject(strings.ToLower(input.Request.Kind.Kind), input.Request.Object.Raw)
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

	out, err := v.evaluateWorkloadMutation(ctx, workload, input, v.metricsEnabled)
	if err != nil {
		slog.Error("error encountered while validating workload", "error", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)

		return
	}

	sendAdmissionReviewResponse(writer, out)
}

// evaluateWorkloadMutation validates the workload and returns an AdmissionReview.
func (v *WorkloadMutationHandler) evaluateWorkloadMutation(
	ctx context.Context,
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
	metricsEnabled bool,
) (*admissionv1.AdmissionReview, error) {
	resourceLogger := kubernetes.NewResourceLoggerForWorkload(v.client, workload)

	slog.Info("evaluating mutation on workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())

	// check if namespace is included
	slog.Debug("checking included namespaces")

	if v.includeNamespaces != nil && len(*v.includeNamespaces) > 0 && !slices.Contains(*v.includeNamespaces, workload.GetNamespace()) {
		slog.Info(
			"workload namespace is not in the list of included namespaces, excluding it from downscaling",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", v.dryRun,
		)

		v.admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(
			review.Request.UID, true,
			http.StatusAccepted,
			"workload namespace is not in the list of included namespaces, excluding it from downscaling",
			v.dryRun), nil
	}

	// check if workload is externally managed
	workloadArray := []scalable.Workload{workload}

	slog.Debug("checking external scaling conditions")

	externalScalingReview, err := v.evaluateWorkloadExternalScalingCondition(ctx, workload, *review)
	if !errors.Is(err, ErrNoExternalScaling) {
		slog.Info("workload is controlled by keda scaledobjects, excluding it")
		v.admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return externalScalingReview, err
	}

	slog.Debug("checking labels, excluded namespaces and excluded workloads")

	workloads := scalable.FilterExcluded(workloadArray, *v.includeLabels, *v.excludeNamespaces, *v.excludeWorkloads, nil)

	if len(workloads) == 0 {
		slog.Info(
			"workload is excluded from downscaling",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", v.dryRun,
		)

		v.admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(
			review.Request.UID, true, http.StatusAccepted, "workload is excluded from downscaling, doesn't need mutation", v.dryRun), nil
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
	if err = scopeWorkload.GetScopeFromAnnotations(workload.GetAnnotations(), resourceLogger, ctx); err != nil {
		slog.Debug("failed to parse workload scope from annotations",
			"error", err,
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", v.dryRun,
		)

		v.admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, true, workload.GetNamespace())

		return newReviewResponse(
			review.Request.UID,
			false,
			http.StatusInternalServerError,
			"failed to parse workload scope from annotations",
			v.dryRun), err
	}

	slog.Debug(
		"parsing namespace scope from workload",
		"name", workload.GetName(),
		"namespace", workload.GetNamespace(),
	)

	scopeNamespace, err := v.client.GetNamespaceScope(workload.GetNamespace(), ctx)
	if err != nil {
		slog.Debug("failed to parse namespace scope from annotations",
			"error", err,
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", v.dryRun,
		)

		v.admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, true, workload.GetNamespace())

		return newReviewResponse(
			review.Request.UID,
			false,
			http.StatusInternalServerError,
			"failed to get namespace scope from annotations",
			v.dryRun), err
	}

	scopes := values.Scopes{scopeWorkload, scopeNamespace, v.scopeCli, v.scopeEnv, v.scopeDefault}

	slog.Debug("finished parsing all scopes", "scopes", scopes, "workload", workload.GetName(), "namespace", workload.GetNamespace())

	if scopes.GetExcluded() {
		slog.Info("workload is excluded from mutation",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", v.dryRun)

		v.admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(
			review.Request.UID,
			true,
			http.StatusAccepted,
			"workload is excluded from downscaling, doesn't need mutation",
			v.dryRun), nil
	}

	scaling := scopes.GetCurrentScaling()

	response, err := evaluateWorkloadScalingConditions(scaling, workload, scopes, review, v.dryRun, metricsEnabled, v.admissionMetrics)
	if err != nil {
		return response, err
	}

	return response, nil
}

// evaluateWorkloadScalingConditions scales the given workload according to the given wanted scaling state.
func evaluateWorkloadScalingConditions(
	scaling values.Scaling,
	workload scalable.Workload,
	scopes values.Scopes,
	review *admissionv1.AdmissionReview,
	dryRun bool,
	metricsEnabled bool,
	admissionMetrics *metrics.AdmissionMetrics,
) (*admissionv1.AdmissionReview, error) {
	if scaling == values.ScalingNone {
		slog.Debug("scaling is not set by any scope, skipping",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", dryRun,
		)

		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(review.Request.UID, true, http.StatusAccepted, "scaling configuration is not set for workload", dryRun), nil
	}

	if scaling == values.ScalingIgnore {
		slog.Debug("scaling is ignored, skipping",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", dryRun,
		)

		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(review.Request.UID, true, http.StatusAccepted, "scaling configuration is ignored for workload", dryRun), nil
	}

	if scaling == values.ScalingMultiple {
		err := newScalingInvalidError(
			`scaling values matched to multiple states.
this is the result of a faulty configuration where on a scope there is multiple values with the same priority
setting different scaling states at the same time (e.g. downtime-period and uptime-period or force-downtime and force-uptime)`,
		)

		slog.Debug("scaling configuration is invalid for workload",
			"error", err, "workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", dryRun,
		)

		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(review.Request.UID, false, http.StatusAccepted, "scaling configuration is invalid for workload", dryRun), err
	}

	if scaling == values.ScalingDown {
		slog.Info("mutating workload matching scaling down condition",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", dryRun,
		)

		downscaleReplicas, err := scopes.GetDownscaleReplicas()
		if err != nil {
			slog.Debug("failed to get downscale replicas from scopes",
				"error", err,
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
				"dryRun", dryRun)

			admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

			return newReviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to get downscaleReplicas", dryRun), err
		}

		response, err := mutateWorkload(workload, review, downscaleReplicas, dryRun, metricsEnabled, admissionMetrics)
		if err != nil {
			return response, err
		}

		return response, nil
	}

	if scaling == values.ScalingUp {
		slog.Debug("workload matches scaling up conditions, skipping",
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", dryRun)

		return newReviewResponse(review.Request.UID, true, http.StatusAccepted, "workload matches scaling up conditions", dryRun), nil
	}

	slog.Debug("workload doesn't match any scaling condition, skipping",
		"workload", workload.GetName(),
		"namespace", workload.GetNamespace(),
		"dryRun", dryRun)

	return newReviewResponse(review.Request.UID, true, http.StatusAccepted, "workload doesn't match any scaling condition", dryRun), nil
}

// evaluateWorkloadExternalScalingCondition checks if the workload is externally managed.
func (v *WorkloadMutationHandler) evaluateWorkloadExternalScalingCondition(
	ctx context.Context,
	workload scalable.Workload,
	review admissionv1.AdmissionReview,
) (*admissionv1.AdmissionReview, error) {
	if _, ok := v.includeResourcesSet["scaledobjects"]; !ok {
		return nil, ErrNoExternalScaling
	}

	scaledObjects, err := v.client.GetScaledObjects(workload.GetNamespace(), ctx)
	if err != nil {
		slog.Error("failed to get scaledobjects from namespace",
			"error", err,
			"namespace", workload.GetNamespace(),
			"workload", workload.GetName(),
			"dryRun", v.dryRun,
		)

		return newReviewResponse(
			review.Request.UID,
			false,
			http.StatusInternalServerError,
			"failed to get scaledobjects from namespace",
			v.dryRun,
		), err
	}

	if scalable.IsWorkloadExternallyManaged(workload, scaledObjects) {
		return newReviewResponse(
			review.Request.UID,
			true,
			http.StatusAccepted,
			"workload is excluded from downscaling, externally managed",
			v.dryRun,
		), nil
	}

	return nil, ErrNoExternalScaling
}

// mutateWorkload mutates the workload by scaling it down based on the scopes.
func mutateWorkload(
	workload scalable.Workload,
	review *admissionv1.AdmissionReview,
	downscaleReplicas values.Replicas,
	dryRun bool,
	metricsEnabled bool,
	admissionMetrics *metrics.AdmissionMetrics,
) (*admissionv1.AdmissionReview, error) {
	// generate a deep copy of the workload to be able to generate a comparison patch
	workloadCopy, err := workload.Copy()
	if err != nil {
		slog.Error("failed to deep copy workload",
			"error", err,
			"workload", workload.GetName(),
			"namespace", workload.GetNamespace(),
			"dryRun", dryRun)

		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

		return newReviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to deep copy workload", dryRun), err
	}

	_, err = workloadCopy.ScaleDown(downscaleReplicas)
	if err != nil {
		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, true, workload.GetNamespace())
		return newReviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to scale down workload", dryRun), err
	}

	patch, err := workload.Compare(workloadCopy)
	if err != nil {
		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, true, workload.GetNamespace())
		return newReviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to compare workload", dryRun), err
	}

	slog.Debug("comparison patch correctly generated", "patch", patch.String())

	// convert the patch into JSON format
	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, true, workload.GetNamespace())
		return newReviewResponse(review.Request.UID, false, http.StatusInternalServerError, "failed to marshal patch", dryRun), err
	}

	if !dryRun {
		admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, true, false, workload.GetNamespace())
		return newPatchReviewResponse(review.Request.UID, jsonPatch)
	}

	admissionMetrics.UpdateValidateWorkloadAdmissionRequestsTotal(metricsEnabled, false, false, workload.GetNamespace())

	return newReviewResponse(review.Request.UID, true, http.StatusAccepted, "would have patched workload", dryRun), nil
}
