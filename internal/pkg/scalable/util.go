package scalable

import (
	"fmt"
	"hash/fnv"
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
	hashedKedaWorkloads := make(map[uint64]bool)
	for _, workload := range workloads {
		slog.Debug("Scanning workload", "kind", workload.GetObjectKind().GroupVersionKind().Kind, "workload", workload.GetName(), "namespace", workload.GetNamespace())
		copiedWorkload := workload
		_ = hashIfScaledObject(copiedWorkload, hashedKedaWorkloads)
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
		if workload.GetObjectKind().GroupVersionKind().Kind != "ScaledObject" {
			if isManagedByKeda(workload, hashedKedaWorkloads) {
				slog.Debug("workload managed by keda scaled object, excluding it from being scanned", "workload", workload.GetName(), "namespace", workload.GetNamespace())
			}
		}
		results = append(results, workload)
	}
	return results
}

// Function to check if the workload is a ScaledObject and update the hash map
func hashIfScaledObject(workload Workload, hashedKedaWorkloads map[uint64]bool) error {
	if workload.GetObjectKind().GroupVersionKind().Kind == "ScaledObject" {
		if replicaWorkload, ok := workload.(*replicaScaledWorkload); ok {
			if scaledObj, ok := replicaWorkload.replicaScaledResource.(*scaledObject); ok {
				targetRefKind, err := scaledObj.getTargetRefKind()
				if err != nil {
					slog.Debug("error getting targetRefKind for scaled object:", "err", err)
				}
				targetRefName, err := scaledObj.getTargetRefName()
				if err != nil {
					slog.Debug("error getting targetRefName for scaled object: ", "err", err)
				}
				targetRefNamespace := scaledObj.GetNamespace()
				computedHash, err := computeHash(targetRefKind, targetRefName, targetRefNamespace)
				if err != nil {
					slog.Debug("error computing hash for scaled object: ", "err", err)
				}
				// store the hash in the map
				hashedKedaWorkloads[computedHash] = true
			} else {
				slog.Debug("replicaScaledResource is not of type *scaledObject")
			}
		} else {
			slog.Debug("workload is not of type *replicaScaledWorkload")
		}
	}
	return nil
}

// computeHash computes a 64-bit FNV-1a hash for the given kind, name, and namespace.
func computeHash(kind string, name string, namespace string) (uint64, error) {
	slog.Debug(fmt.Sprintf("generating hash for values %s:%s:%s", kind, name, namespace))

	hash := fnv.New64a()
	_, err := hash.Write([]byte(fmt.Sprintf("%s:%s:%s", kind, name, namespace)))
	if err != nil {
		return 0, fmt.Errorf("failed to write to hash: %w", err)
	}

	computedHash := hash.Sum64()

	return computedHash, nil
}

func isManagedByKeda(workload Workload, hashedKedaWorkloads map[uint64]bool) bool {
	kind := workload.GetObjectKind().GroupVersionKind().Kind
	name := workload.GetName()
	namespace := workload.GetNamespace()

	if kind == "" {
		slog.Warn("warning: kind is empty!")
	}

	computedHash, err := computeHash(kind, name, namespace)
	if err != nil {
		slog.Debug("error computing hash for workload: ", "err", err)
		return false
	}

	if _, exists := hashedKedaWorkloads[computedHash]; exists {
		return true
	}
	return false
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
