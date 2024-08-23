package scalable

import (
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

func FilterMatchingLabels(workloads []Workload, includeLabels values.RegexList) []Workload {
	var results []Workload
	if includeLabels == nil {
		results = append(results, workloads...)
		return results
	}
	for _, workload := range workloads {
		for label, value := range workload.GetLabels() {
			if includeLabels.CheckMatchesAny(fmt.Sprintf("%s=%s", label, value)) {
				results = append(results, workload)
			}
		}
	}
	return results
}
