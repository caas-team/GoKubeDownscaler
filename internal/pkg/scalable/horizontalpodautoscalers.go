package scalable

import (
	"context"
	"fmt"
	"math"

	appsv1 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// getHorizontalPodAutoscalers is the getResourceFunc for horizontalPodAutoscaler
func getHorizontalPodAutoscalers(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	poddisruptionbudgets, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get horizontalpodautoscalers: %w", err)
	}
	for _, item := range poddisruptionbudgets.Items {
		results = append(results, horizontalPodAutoscaler{&item})
	}
	return results, nil
}

// horizontalPodAutoscaler is a wrapper for autoscaling/v2.horizontalPodAutoscaler to implement the scalableResource interface
type horizontalPodAutoscaler struct {
	*appsv1.HorizontalPodAutoscaler
}

// SetMinReplicas set the spec.MinReplicas to a new value
func (h horizontalPodAutoscaler) SetMinReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < math.MinInt32 {
		return fmt.Errorf("replicas value exceeds int32 bounds")
	}

	// #nosec G115
	newReplicas := int32(replicas)
	h.Spec.MinReplicas = &newReplicas
	return nil
}

// GetMinReplicas get the spec.MinReplicas from the resource
func (h horizontalPodAutoscaler) GetMinReplicas() (int, error) {
	minReplicas := h.Spec.MinReplicas
	if minReplicas == nil {
		return 0, errNoMinReplicasSpecified
	}
	return int(*minReplicas), nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (h horizontalPodAutoscaler) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.AutoscalingV2().HorizontalPodAutoscalers(h.Namespace).Update(ctx, h.HorizontalPodAutoscaler, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update horizontalpodautoscaler: %w", err)
	}
	return nil
}
