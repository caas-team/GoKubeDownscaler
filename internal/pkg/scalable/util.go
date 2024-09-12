package scalable

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
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
	return results
}

// isMatchingLabels check if the workload is matching any labels
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

// isNamespaceExcluded check if the workloads namespace is excluded
func isNamespaceExcluded(workload Workload, excludedNamespaces values.RegexList) bool {
	if excludedNamespaces == nil {
		return false
	}
	return excludedNamespaces.CheckMatchesAny(workload.GetNamespace())
}

// isWorkloadExcluded check if the workloads name is excluded
func isWorkloadExcluded(workload Workload, excludedWorkloads values.RegexList) bool {
	if excludedWorkloads == nil {
		return false
	}
	return excludedWorkloads.CheckMatchesAny(workload.GetName())
}

// setOriginalReplicas sets the original replicas annotation on the workload
func setOriginalReplicas(originalReplicas int, workload Workload) {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[annotationOriginalReplicas] = strconv.Itoa(originalReplicas)
	workload.SetAnnotations(annotations)
}

// getOriginalReplicas gets the original replicas annotation on the workload. nil is undefined
func getOriginalReplicas(workload Workload) (*int, error) {
	annotations := workload.GetAnnotations()
	originalReplicasString, ok := annotations[annotationOriginalReplicas]
	if !ok {
		return nil, nil
	}
	originalReplicas, err := strconv.Atoi(originalReplicasString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original replicas annotation on workload: %w", err)
	}
	return &originalReplicas, nil
}

// removeOriginalReplicas removes the annotationOriginalReplicas from the workload
func removeOriginalReplicas(workload Workload) {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
}
