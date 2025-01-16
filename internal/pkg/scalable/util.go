package scalable

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
)

const (
	annotationOriginalReplicas = "downscaler/original-replicas"
)

// FilterExcluded filters the workloads to match the includeLabels, excludedNamespaces and excludedWorkloads.
func FilterExcluded(workloads []Workload, includeLabels, excludedNamespaces, excludedWorkloads util.RegexList) []Workload {
	results := make([]Workload, 0, len(workloads))

	for _, workload := range workloads {
		if !isMatchingLabels(workload, includeLabels) {
			slog.Debug(
				"workload is not matching any of the specified labels, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)

			continue
		}

		if isNamespaceExcluded(workload, excludedNamespaces) {
			slog.Debug(
				"the workloads namespace is excluded, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)

			continue
		}

		if isWorkloadExcluded(workload, excludedWorkloads) {
			slog.Debug(
				"the workloads name is excluded, excluding it from being scanned",
				"workload", workload.GetName(),
				"namespace", workload.GetNamespace(),
			)

			continue
		}

		results = append(results, workload)
	}

	return results[:len(results):len(results)] // unallocate excess capacity
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
func isWorkloadExcluded(workload Workload, excludedWorkloads util.RegexList) bool {
	if excludedWorkloads == nil {
		return false
	}

	return excludedWorkloads.CheckMatchesAny(workload.GetName())
}

// setOriginalReplicas sets the original replicas annotation on the workload.
func setOriginalReplicas(originalReplicas int32, workload Workload) {
	annotations := workload.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[annotationOriginalReplicas] = strconv.Itoa(int(originalReplicas))
	workload.SetAnnotations(annotations)
}

// getOriginalReplicas gets the original replicas annotation on the workload. nil is undefined.
func getOriginalReplicas(workload Workload) (*int32, error) {
	annotations := workload.GetAnnotations()

	originalReplicasString, ok := annotations[annotationOriginalReplicas]
	if !ok {
		return nil, nil //nolint: nilnil // should get fixed along with https://github.com/caas-team/GoKubeDownscaler/issues/7
	}

	originalReplicas, err := strconv.ParseInt(originalReplicasString, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original replicas annotation on workload: %w", err)
	}

	// #nosec G115
	result := int32(originalReplicas)

	return &result, nil
}

// removeOriginalReplicas removes the annotationOriginalReplicas from the workload.
func removeOriginalReplicas(workload Workload) {
	annotations := workload.GetAnnotations()
	delete(annotations, annotationOriginalReplicas)
	workload.SetAnnotations(annotations)
}
