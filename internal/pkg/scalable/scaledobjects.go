package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

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
	slog.Error("error before fetching")
	if err != nil {
		slog.Error("error after fetching")
		return nil, fmt.Errorf("failed to get scaledobjects: %w", err)
	}
	for _, item := range scaledobjects.Items {
		slog.Error("error inside foreach")
		so := &kedav1alpha1.ScaledObject{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, so); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to scaledobject: %w", err)
		}
		results = append(results, ScaledObject{so})
	}
	slog.Error("no error")
	return results, nil
}

// ScaledObject is a wrapper for keda.sh/v1alpha1.horizontalPodAutoscaler to implement the scalableResource interface
type ScaledObject struct {
	*kedav1alpha1.ScaledObject
}

// GetPauseScaledObjectAnnotationReplicasIfExistsAndValid gets the value of keda pause annotations. It returns the int value and true if the annotations exists and it is well formatted, otherwise it returns a fake value and false
func (s ScaledObject) GetPauseScaledObjectAnnotationReplicasIfExistsAndValid() (int, bool, error) {
	if pausedReplicasStr, ok := s.Annotations[kedaPausedReplicasAnnotation]; ok {
		pausedReplicas, err := strconv.Atoi(pausedReplicasStr)
		if err != nil {
			return 1, false, fmt.Errorf("invalid value for annotation %s: %w", kedaPausedReplicasAnnotation, err)
		}
		return pausedReplicas, true, nil
	}

	return 1, false, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s ScaledObject) Update(_ *kubernetes.Clientset, dynamicClient dynamic.Interface, ctx context.Context) error {
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s.ScaledObject)
	if err != nil {
		return fmt.Errorf("failed to convert scaledobject to unstructured: %w", err)
	}
	unstructuredResource := &unstructured.Unstructured{Object: unstructuredObj}
	_, err = dynamicClient.Resource(kedav1alpha1.SchemeGroupVersion.WithResource("scaledobjects")).Namespace(s.Namespace).Update(ctx, unstructuredResource, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ScaledObject: %w", err)
	}

	return nil
}
