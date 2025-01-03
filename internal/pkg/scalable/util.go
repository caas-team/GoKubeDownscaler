package scalable

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	annotationOriginalReplicas = "downscaler/original-replicas"
)

// FilterExcluded filters the workloads to match the includeLabels, excludedNamespaces and excludedWorkloads
func FilterExcluded(workloads []Workload, includeLabels values.RegexList, excludedNamespaces values.RegexList, excludedWorkloads values.RegexList) []Workload {
	var results []Workload
	for _, workload := range workloads {
		if !isMatchingLabels(workload, includeLabels) {
			slog.Debug("workload is not matching any of the specified labels, excluding it from being scanned", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			continue
		}
		if isNamespaceExcluded(workload, excludedNamespaces) {
			slog.Debug("the workloads namespace is excluded, excluding it from being scanned", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			continue
		}
		if isWorkloadExcluded(workload, excludedWorkloads) {
			slog.Debug("the workloads name is excluded, excluding it from being scanned", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			continue
		}
		results = append(results, workload)
	}
	results = filterExternallyScaled(results)
	return results
}

type workloadIdentifier struct {
	gvk       schema.GroupVersionKind
	name      string
	namespace string
}

// filterExternallyScaled filters out workloads which are scaled via external sources like a scaledObject
func filterExternallyScaled(workloads []Workload) []Workload {
	var excludedWorkloads []workloadIdentifier
	var result []Workload
	for _, workload := range workloads {
		scaledobject := getWorkloadAsScaledObject(workload)
		if scaledobject == nil {
			continue
		}

		excludedWorkloads = append(excludedWorkloads, workloadIdentifier{
			gvk: schema.GroupVersionKind{
				Kind:    scaledobject.Spec.ScaleTargetRef.Kind,
				Group:   strings.Split(scaledobject.Spec.ScaleTargetRef.APIVersion, "/")[0],
				Version: strings.Split(scaledobject.Spec.ScaleTargetRef.APIVersion, "/")[1],
			},
			name:      scaledobject.Spec.ScaleTargetRef.Name,
			namespace: scaledobject.Namespace,
		})
	}
	for _, workload := range workloads {
		if !slices.ContainsFunc(excludedWorkloads, func(wi workloadIdentifier) bool { return doesWorkloadMatchIdentifier(workload, wi) }) {
			continue
		}
		result = append(result, workload)
	}
	return result
}

// doesWorkloadMatchIdentifier checks if the workload matches the given workload identifier
func doesWorkloadMatchIdentifier(workload Workload, wi workloadIdentifier) bool {
	if wi.name != workload.GetName() {
		return false
	}
	if wi.namespace != workload.GetNamespace() {
		return false
	}
	if !(wi.gvk.Group == "" || wi.gvk.Group == workload.GroupVersionKind().Group) {
		return false
	}
	if !(wi.gvk.Version == "" || wi.gvk.Version == workload.GroupVersionKind().Version) {
		return false
	}
	if !(wi.gvk.Kind == "" || wi.gvk.Kind == workload.GroupVersionKind().Kind) {
		return false
	}
	return true
}

// getWorkloadAsScaledObject tries to get the given workload as an scaled object
func getWorkloadAsScaledObject(workload Workload) *scaledObject {
	replicaScaled, ok := workload.(*replicaScaledWorkload)
	if !ok {
		return nil
	}
	scaledobject, ok := replicaScaled.replicaScaledResource.(*scaledObject)
	if !ok {
		return nil
	}
	return scaledobject
}

// isMatchingLabels check if the workload is matching any of the specified labels
func isMatchingLabels(workload Workload, includeLabels values.RegexList) bool {
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

// isNamespaceExcluded checks if the workloads namespace is excluded
func isNamespaceExcluded(workload Workload, excludedNamespaces values.RegexList) bool {
	if excludedNamespaces == nil {
		return false
	}
	return excludedNamespaces.CheckMatchesAny(workload.GetNamespace())
}

// isWorkloadExcluded checks if the workloads name is excluded
func isWorkloadExcluded(workload Workload, excludedWorkloads values.RegexList) bool {
	if excludedWorkloads == nil {
		return false
	}
	return excludedWorkloads.CheckMatchesAny(workload.GetName())
}

// setOriginalReplicas sets the original replicas annotation on the workload
func setOriginalReplicas(originalReplicas int32, workload Workload) {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[annotationOriginalReplicas] = strconv.Itoa(int(originalReplicas))
	workload.SetAnnotations(annotations)
}

// getOriginalReplicas gets the original replicas annotation on the workload. nil is undefined
func getOriginalReplicas(workload Workload) (*int32, error) {
	annotations := workload.GetAnnotations()
	originalReplicasString, ok := annotations[annotationOriginalReplicas]
	if !ok {
		return nil, nil
	}
	originalReplicas, err := strconv.ParseInt(originalReplicasString, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original replicas annotation on workload: %w", err)
	}
	// #nosec G115
	result := int32(originalReplicas)
	return &result, nil
}

// removeOriginalReplicas removes the annotationOriginalReplicas from the workload
func removeOriginalReplicas(workload Workload) {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
}
