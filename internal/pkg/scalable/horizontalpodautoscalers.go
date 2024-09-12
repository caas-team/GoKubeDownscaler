package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	appsv1 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getHorizontalPodAutoscalers is the getResourceFunc for horizontalPodAutoscalers
func getHorizontalPodAutoscalers(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	hpas, err := clientsets.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get horizontalpodautoscalers: %w", err)
	}
	for _, item := range hpas.Items {
		results = append(results, &horizontalPodAutoscaler{&item})
	}
	return results, nil
}

// horizontalPodAutoscaler is a wrapper for autoscaling/v2.horizontalPodAutoscaler to implement the Workload interface
type horizontalPodAutoscaler struct {
	*appsv1.HorizontalPodAutoscaler
}

// setMinReplicas set the spec.MinReplicas to a new value
func (h *horizontalPodAutoscaler) setMinReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < 1 {
		return errBoundOnScalingTargetValue
	}

	// #nosec G115
	newReplicas := int32(replicas)
	h.Spec.MinReplicas = &newReplicas
	return nil
}

// getMinReplicas gets the spec.MinReplicas from the resource
func (h *horizontalPodAutoscaler) getMinReplicas() (int, error) {
	minReplicas := h.Spec.MinReplicas
	if minReplicas == nil {
		return 0, errNoMinReplicasSpecified
	}
	return int(*minReplicas), nil
}

// ScaleUp scales the resource up
func (h *horizontalPodAutoscaler) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(h)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == nil {
		slog.Debug("original replicas is not set, skipping", "workload", h.GetName(), "namespace", h.GetNamespace())
		return nil
	}

	err = h.setMinReplicas(*originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set minimum replicas for workload: %w", err)
	}
	removeOriginalReplicas(h)
	return nil
}

// ScaleDown scales the resource down
func (h *horizontalPodAutoscaler) ScaleDown(downscaleReplicas int) error {
	originalReplicas, err := h.getMinReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already scaled down, skipping", "workload", h.GetName(), "namespace", h.GetNamespace())
		return nil
	}

	err = h.setMinReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set min replicas for workload: %w", err)
	}
	setOriginalReplicas(originalReplicas, h)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (h *horizontalPodAutoscaler) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(h.Namespace).Update(ctx, h.HorizontalPodAutoscaler, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update horizontalpodautoscaler: %w", err)
	}
	return nil
}
