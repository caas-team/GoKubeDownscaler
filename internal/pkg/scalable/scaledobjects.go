package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	kedaPausedReplicasAnnotation = "autoscaling.keda.sh/paused-replicas"
)

// getScaledObjects is the getResourceFunc for Keda ScaledObjects
func getScaledObjects(namespace string, _ *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	scaledobjects, err := dynamicClient.Resource(kedav1alpha1.SchemeGroupVersion.WithResource("scaledobjects")).Namespace(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get scaledobjects: %w", err)
	}
	for _, item := range scaledobjects.Items {
		so := &kedav1alpha1.ScaledObject{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, so); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to scaledobject: %w", err)
		}
		results = append(results, &scaledObject{so})
	}
	return results, nil
}

// scaledObject is a wrapper for keda.sh/v1alpha1.horizontalPodAutoscaler to implement the Workload interface
type scaledObject struct {
	*kedav1alpha1.ScaledObject
}

// getPauseScaledObjectAnnotationReplicasIfExistsAndValid gets the value of keda pause annotations. It returns the int value and true if the annotations exists and it is well formatted, otherwise it returns a fake value and false
func (s *scaledObject) getPauseScaledObjectAnnotationReplicasIfExistsAndValid() (int, bool, error) {
	if pausedReplicasStr, ok := s.Annotations[kedaPausedReplicasAnnotation]; ok {
		pausedReplicas, err := strconv.Atoi(pausedReplicasStr)
		if err != nil {
			return 1, false, fmt.Errorf("invalid value for annotation %s: %w", kedaPausedReplicasAnnotation, err)
		}
		return pausedReplicas, true, nil
	}

	return 1, false, nil
}

// ScaleUp upscale the resource
func (s *scaledObject) ScaleUp() error {
	_, pauseAnnotationExists, err := s.getPauseScaledObjectAnnotationReplicasIfExistsAndValid()
	if err != nil {
		return fmt.Errorf("failed to get pause scaledobject annotation: %w", err)
	}
	if !pauseAnnotationExists {
		return fmt.Errorf("the workload is already upscaled: %w", err)
	}

	originalReplicas, err := getOriginalReplicas(s)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == values.Undefined {
		slog.Debug("original replicas is not set, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}

	removeOriginalReplicas(s)
	delete(s.GetAnnotations(), kedaPausedReplicasAnnotation)
	return nil
}

// ScaleDown downscale the resource
func (s *scaledObject) ScaleDown(downscaleReplicas int) error {
	pauseAnnotationReplicas, pauseAnnotationExists, err := s.getPauseScaledObjectAnnotationReplicasIfExistsAndValid()
	if err != nil {
		return fmt.Errorf("failed to get pause scaledobject annotation: %w", err)
	}
	if pauseAnnotationExists && pauseAnnotationReplicas == downscaleReplicas {
		setOriginalReplicas(downscaleReplicas, s)
		return nil
	}
	if (pauseAnnotationExists && pauseAnnotationReplicas != downscaleReplicas) || !pauseAnnotationExists {
		replicasStr := strconv.Itoa(downscaleReplicas)
		setOriginalReplicas(downscaleReplicas, s)
		annotations := s.GetAnnotations()
		annotations[kedaPausedReplicasAnnotation] = replicasStr
		return nil
	}
	return fmt.Errorf("invalid downscaling case for scaledobject")
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *scaledObject) Update(_ *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) error {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s.ScaledObject)
	if err != nil {
		return fmt.Errorf("failed to convert scaledobject to unstructured: %w", err)
	}
	unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
	_, err = dynamicClient.Resource(kedav1alpha1.SchemeGroupVersion.WithResource("scaledobjects")).Namespace(s.Namespace).Update(ctx, unstructuredResource, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update scaledObject: %w", err)
	}

	return nil
}
