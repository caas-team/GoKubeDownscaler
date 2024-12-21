package scalable

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"

	appsv1 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errMinReplicasBoundsExceeded = errors.New("error: a HPAs minReplicas can only be set to int32 values larger than 1")

// Define the GVK for horizontalPodAutoscalers
var horizontalPodAutoscalerGVK = schema.GroupVersionKind{
	Group:   "autoscaling",
	Version: "v2",
	Kind:    "HorizontalPodAutoscaler",
}

// getHorizontalPodAutoscalers is the getResourceFunc for horizontalPodAutoscalers
func getHorizontalPodAutoscalers(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var results []Workload
	hpas, err := clientsets.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get horizontalpodautoscalers: %w", err)
	}
	for _, item := range hpas.Items {
		results = append(results, &replicaScaledWorkload{&horizontalPodAutoscaler{&item}})
	}
	return results, nil
}

// horizontalPodAutoscaler is a wrapper for autoscaling/v2.HorizontalPodAutoscaler to implement the replicaScaledResource interface
type horizontalPodAutoscaler struct {
	*appsv1.HorizontalPodAutoscaler
}

// GetObjectKind sets the GVK for horizontalPodAutoscaler
func (h *horizontalPodAutoscaler) GetObjectKind() schema.ObjectKind {
	return h
}

// GroupVersionKind returns the GVK for horizontalPodAutoscaler
func (h *horizontalPodAutoscaler) GroupVersionKind() schema.GroupVersionKind {
	return horizontalPodAutoscalerGVK
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called
func (h *horizontalPodAutoscaler) setReplicas(replicas int32) error {
	if replicas < 1 {
		return errMinReplicasBoundsExceeded
	}
	h.Spec.MinReplicas = &replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource
func (h *horizontalPodAutoscaler) getReplicas() (int32, error) {
	replicas := h.Spec.MinReplicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return *h.Spec.MinReplicas, nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (h *horizontalPodAutoscaler) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(h.Namespace).Update(ctx, h.HorizontalPodAutoscaler, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update horizontalpodautoscaler: %w", err)
	}
	return nil
}
