package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	annotationKedaPausedReplicas = "autoscaling.keda.sh/paused-replicas"
)

// getScaledObjects is the getResourceFunc for Keda ScaledObjects
func getScaledObjects(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	scaledobjects, err := clientsets.Keda.KedaV1alpha1().ScaledObjects(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get scaledobjects: %w", err)
	}
	for _, item := range scaledobjects.Items {
		results = append(results, &scaledObject{&item})
	}
	return results, nil
}

// scaledObject is a wrapper for keda.sh/v1alpha1.horizontalPodAutoscaler to implement the Workload interface
type scaledObject struct {
	*kedav1alpha1.ScaledObject
}

// getPauseAnnotation gets the value of keda pause annotations
func (s *scaledObject) getPauseAnnotation() (int, error) {
	pausedReplicasAnnotation, ok := s.Annotations[annotationKedaPausedReplicas]
	if !ok {
		return values.Undefined, nil
	}
	pausedReplicas, err := strconv.Atoi(pausedReplicasAnnotation)
	if err != nil {
		return 0, fmt.Errorf("invalid value for annotation %s: %w", annotationKedaPausedReplicas, err)
	}
	return pausedReplicas, nil
}

// setPauseAnnotation sets the value of keda pause annotations
func (s *scaledObject) setPauseAnnotation(value int) {
	if s.Annotations == nil {
		s.Annotations = map[string]string{}
	}
	s.Annotations[annotationKedaPausedReplicas] = strconv.Itoa(value)
}

// ScaleUp upscale the resource when the downscale period ends
func (s *scaledObject) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(s)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}
	if *originalReplicas == values.Undefined { // pausedAnnotation was not defined before workload was downscaled
		delete(s.Annotations, annotationKedaPausedReplicas)
		removeOriginalReplicas(s)
		return nil
	}
	s.setPauseAnnotation(*originalReplicas)
	removeOriginalReplicas(s)
	return nil
}

// ScaleDown scales down the workload
func (s *scaledObject) ScaleDown(downscaleReplicas int) error {
	pausedReplicas, err := s.getPauseAnnotation()
	if err != nil {
		return fmt.Errorf("failed to get pause scaledobject annotation: %w", err)
	}
	if pausedReplicas == downscaleReplicas {
		slog.Debug("workload is already scaled down, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}
	s.setPauseAnnotation(downscaleReplicas)
	setOriginalReplicas(pausedReplicas, s)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *scaledObject) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Keda.KedaV1alpha1().ScaledObjects(s.Namespace).Update(ctx, s.ScaledObject, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update scaledObject: %w", err)
	}

	return nil
}
