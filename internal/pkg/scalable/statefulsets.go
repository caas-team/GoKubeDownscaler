package scalable

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"

	"k8s.io/client-go/dynamic"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// getStatefulSets is the getResourceFunc for StatefulSets
func getStatefulSets(namespace string, clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) ([]Workload, error) {
	var results []Workload
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{TimeoutSeconds: &timeout})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}
	for _, item := range statefulsets.Items {
		results = append(results, &statefulSet{&item})
	}
	return results, nil
}

// statefulset is a wrapper for appsv1.statefulSet to implement the Workload interface
type statefulSet struct {
	*appsv1.StatefulSet
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on kubernetes until update() is called
func (s *statefulSet) setReplicas(replicas int) error {
	if replicas > math.MaxInt32 || replicas < 0 {
		return errBoundOnScalingTargetValue
	}

	// #nosec G115
	newReplicas := int32(replicas)
	s.Spec.Replicas = &newReplicas
	return nil
}

// getCurrentReplicas gets the current amount of replicas of the resource
func (s *statefulSet) getCurrentReplicas() (int, error) {
	replicas := s.Spec.Replicas
	if replicas == nil {
		return 0, errNoReplicasSpecified
	}
	return int(*s.Spec.Replicas), nil
}

// ScaleUp upscale the resource when the downscale period ends
func (s *statefulSet) ScaleUp() error {
	originalReplicas, err := getOriginalReplicas(s)
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == values.Undefined {
		slog.Debug("original replicas is not set, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}

	err = s.setReplicas(originalReplicas)
	if err != nil {
		return fmt.Errorf("failed to set original replicas for workload: %w", err)
	}
	removeOriginalReplicas(s)
	return nil
}

// ScaleDown downscale the resource when the downscale period starts
func (s *statefulSet) ScaleDown(downscaleReplicas int) error {
	originalReplicas, err := s.getCurrentReplicas()
	if err != nil {
		return fmt.Errorf("failed to get original replicas for workload: %w", err)
	}
	if originalReplicas == downscaleReplicas {
		slog.Debug("workload is already at downscale replicas, skipping", "workload", s.GetName(), "namespace", s.GetNamespace())
		return nil
	}

	err = s.setReplicas(downscaleReplicas)
	if err != nil {
		return fmt.Errorf("failed to set replicas for workload: %w", err)
	}
	setOriginalReplicas(originalReplicas, s)
	return nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *statefulSet) Update(clientset *kubernetes.Clientset, _ dynamic.Interface, ctx context.Context) error {
	_, err := clientset.AppsV1().StatefulSets(s.Namespace).Update(ctx, s.StatefulSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}
	return nil
}
