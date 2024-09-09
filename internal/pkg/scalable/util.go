package scalable

import (
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

// FilterExcluded filters the workloads to match the includeLabels, excludedNamespaces and excludedWorkloads
func FilterExcluded(workloads []Workload, includeLabels values.RegexList, excludedNamespaces values.RegexList, excludedWorkloads values.RegexList) []Workload {
	var results []Workload
	for _, workload := range workloads {
		if !isMatchingLabels(workload, includeLabels) {
			continue
		}
		if isNamespaceExcluded(workload, excludedNamespaces) {
			continue
		}
		if isWorkloadExcluded(workload, excludedWorkloads) {
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
	return excludedNamespaces.CheckMatchesAny(workload.GetName())
}

// isWorkloadExcluded check if the workloads name is excluded
func isWorkloadExcluded(workload Workload, excludedWorkloads values.RegexList) bool {
	if excludedWorkloads == nil {
		return false
	}
	return excludedWorkloads.CheckMatchesAny(workload.GetName())
}
