package scalable

import (
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	annotationOriginalReplicas          = "downscaler/original-replicas"
	defaultKedaScaleTargetRefApiVersion = "apps/v1"
	defaultKedaScaleTargetRefKind       = "Deployment"
	kafkaStrimziGroup                   = "kafka.strimzi.io"
	kafkaStrimziVersion                 = "v1"
)

// FilterExcluded filters the workloads to match the includeLabels, excludedNamespaces and excludedWorkloads.
func FilterExcluded(
	workloads []Workload,
	includeLabels,
	excludedNamespaces,
	excludedWorkloads util.RegexList,
	currentNamespaceToMetrics map[string]*metrics.NamespaceMetricsHolder,
) []Workload {
	externallyScaled := getExternallyScaled(workloads)

	results := make([]Workload, 0, len(workloads))

	for _, workload := range workloads {
		if currentNamespaceToMetrics != nil {
			_, ok := currentNamespaceToMetrics[workload.GetNamespace()]
			if !ok {
				namespaceMetrics := metrics.NewNamespaceMetricsHolder()
				currentNamespaceToMetrics[workload.GetNamespace()] = namespaceMetrics
			}
		}

		if !isMatchingLabels(workload, includeLabels) {
			slog.Debug(
				"workload is not matching any of the specified labels, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)
			currentNamespaceToMetrics[workload.GetNamespace()].IncrementExcludedWorkloadsCount()

			continue
		}

		if isNamespaceExcluded(workload, excludedNamespaces) {
			slog.Debug(
				"the workloads namespace is excluded, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)
			currentNamespaceToMetrics[workload.GetNamespace()].IncrementExcludedWorkloadsCount()

			continue
		}

		if isWorkloadExcluded(workload, excludedWorkloads) {
			slog.Debug(
				"the workloads name is excluded, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)
			currentNamespaceToMetrics[workload.GetNamespace()].IncrementExcludedWorkloadsCount()

			continue
		}

		if isExternallyScaled(workload, externallyScaled) {
			slog.Debug(
				"the workload is scaled externally, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)
			currentNamespaceToMetrics[workload.GetNamespace()].IncrementExcludedWorkloadsCount()

			continue
		}

		results = append(results, workload)
	}

	return slices.Clip(results)
}

func IsWorkloadExternallyManaged(workload Workload, workloadsManagers []Workload) bool {
	externallyScaled := getExternallyScaled(workloadsManagers)
	return isExternallyScaled(workload, externallyScaled)
}

type workloadIdentifier struct {
	gvk       schema.GroupVersionKind
	name      string
	namespace string
}

// setGroupVersionKindIfEmpty preserves an existing GVK and only fills it when list/get decoding left TypeMeta empty.
func setGroupVersionKindIfEmpty(obj runtime.Object, gvk schema.GroupVersionKind) {
	if obj == nil {
		return
	}

	if !obj.GetObjectKind().GroupVersionKind().Empty() {
		return
	}

	obj.GetObjectKind().SetGroupVersionKind(gvk)
}

// getExternallyScaled returns identifiers for workloads which are being scaled externally and should therefore be excluded.
func getExternallyScaled(workloads []Workload) []workloadIdentifier {
	externallyScaled := make([]workloadIdentifier, 0, len(workloads))

	for _, workload := range workloads {
		scaledobject := getWorkloadAsScaledObject(workload)
		if scaledobject == nil {
			continue
		}

		var version, group string
		apiVersion := scaledobject.Spec.ScaleTargetRef.APIVersion
		kind := scaledobject.Spec.ScaleTargetRef.Kind

		if apiVersion == "" {
			apiVersion = defaultKedaScaleTargetRefApiVersion
		}

		if kind == "" {
			kind = defaultKedaScaleTargetRefKind
		}

		apiVersionSlice := strings.SplitN(apiVersion, "/", 2)
		if len(apiVersionSlice) < 2 {
			group = ""
			version = apiVersionSlice[0]
		} else {
			group = apiVersionSlice[0]
			version = apiVersionSlice[1]
		}

		externallyScaled = append(externallyScaled, workloadIdentifier{
			gvk: schema.GroupVersionKind{
				Kind:    kind,
				Group:   group,
				Version: version,
			},
			name:      scaledobject.Spec.ScaleTargetRef.Name,
			namespace: scaledobject.Namespace,
		})
	}

	return slices.Clip(externallyScaled)
}

// isExternallyScaled checks if the workload matches any of the given workload identifiers.
func isExternallyScaled(workload Workload, externallyScaled []workloadIdentifier) bool {
	workloadGVK := workload.GroupVersionKind()
	workloadName := workload.GetName()
	workloadNamespace := workload.GetNamespace()

	for i := range externallyScaled {
		if !matchesWorkloadIdentifier(&externallyScaled[i], workloadName, workloadNamespace, workloadGVK) {
			continue
		}

		return true
	}

	return false
}

// matchesWorkloadIdentifier checks if the workload matches the given workload identifier.
// Resources are uniquely identified by group, kind, name, and namespace in Kubernetes.
// Version is not checked because multiple API versions can refer to the same resource,
// so version doesn't affect resource identity.
func matchesWorkloadIdentifier(
	wid *workloadIdentifier,
	workloadName, workloadNamespace string,
	workloadGVK schema.GroupVersionKind,
) bool {
	if wid.name != workloadName || wid.namespace != workloadNamespace {
		return false
	}

	if !matchesGVKField(wid.gvk.Group, workloadGVK.Group, false) {
		return false
	}

	if !matchesGVKField(wid.gvk.Kind, workloadGVK.Kind, true) {
		return false
	}

	return true
}

func matchesGVKField(expected, actual string, caseInsensitive bool) bool {
	if expected == "" || actual == "" {
		return true
	}

	if caseInsensitive {
		return strings.EqualFold(expected, actual)
	}

	return expected == actual
}

// getWorkloadAsScaledObject tries to get the given workload as a scaled object.
func getWorkloadAsScaledObject(workload Workload) *scaledObject {
	replicaScaled, isReplicaScaled := workload.(*replicaScaledWorkload)
	if !isReplicaScaled {
		return nil
	}

	scaledObject, isScaledObject := replicaScaled.replicaScaledResource.(*scaledObject)
	if !isScaledObject {
		return nil
	}

	return scaledObject
}

// isMatchingLabels check if the workload is matching any of the specified labels.
func isMatchingLabels(workload Workload, includeLabels util.RegexList) bool {
	if includeLabels == nil {
		return true
	}

	for label, value := range workload.GetLabels() {
		if !includeLabels.CheckMatchesAny(fmt.Sprintf("%s=%s", label, value)) {
			continue
		}

		return true
	}

	return false
}

// isNamespaceExcluded checks if the workloads namespace is excluded.
func isNamespaceExcluded(workload Workload, excludedNamespaces util.RegexList) bool {
	if excludedNamespaces == nil {
		return false
	}

	return excludedNamespaces.CheckMatchesAny(workload.GetNamespace())
}

// isWorkloadExcluded checks if the workloads name is excluded.
func isWorkloadExcluded(
	workload Workload,
	excludedWorkloads util.RegexList,
) bool {
	if isManagedByOwnerReference(workload) {
		return true
	}

	if excludedWorkloads == nil {
		return false
	}

	return excludedWorkloads.CheckMatchesAny(workload.GetName())
}

// isManagedByOwnerReference checks if the workload is managed by an owner reference that is in the includedResources list.
func isManagedByOwnerReference(workload Workload) bool {
	for _, ownerReference := range workload.GetOwnerReferences() {
		if ownerReference.Controller != nil && *ownerReference.Controller {
			return true
		}
	}

	return false
}

// setOriginalReplicas sets the original replicas annotation on the workload.
func setOriginalReplicas(replicaCount values.Replicas, workload Workload) {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[annotationOriginalReplicas] = replicaCount.String()

	workload.SetAnnotations(annotations)
}

// getOriginalReplicas gets the original replicas annotation on the workload. nil is undefined.
func getOriginalReplicas(workload Workload) (values.Replicas, error) {
	annotations := workload.GetAnnotations()

	originalReplicasString, ok := annotations[annotationOriginalReplicas]
	if !ok {
		return nil, newOriginalReplicasUnsetError("error: original replicas annotation not set on workload")
	}

	var replica values.Replicas
	replicasValue := values.ReplicasValue{Replicas: &replica}

	if err := replicasValue.Set(originalReplicasString); err != nil {
		return nil, fmt.Errorf("failed to parse original replicas annotation on workload: %w", err)
	}

	return replica, nil
}

// removeOriginalReplicas removes the annotationOriginalReplicas from the workload.
func removeOriginalReplicas(workload Workload) {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
}

// derefInt32 safely dereference int32, if not present a default value is set instead.
func derefInt32(p *int32, def int32) int32 {
	if p != nil {
		return *p
	}

	return def
}

// unstructuredReplicasToInt32 converts an unstructured replicas value to int32.
// The Kubernetes JSON decoder returns numbers as float64; int64 is also accepted for robustness.
// Returns false if the type is neither float64 nor int64.
func unstructuredReplicasToInt32(val any) (int32, bool) {
	switch value := val.(type) {
	case int64:
		if value < math.MinInt32 || value > math.MaxInt32 {
			return 0, false
		}

		return int32(value), true

	case float64:
		if value != math.Trunc(value) {
			return 0, false
		}

		if value < math.MinInt32 || value > math.MaxInt32 {
			return 0, false
		}

		return int32(value), true

	default:
		return 0, false
	}
}
