package scalable

import (
	"fmt"
	"strconv"
)

const (
	annotationOriginalReplicas = "downscaler/original-replicas"
)

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
