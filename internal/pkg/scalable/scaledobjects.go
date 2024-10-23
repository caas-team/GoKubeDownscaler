package scalable

import (
	"context"
	"fmt"
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
		results = append(results, &replicaScaledWorkload{&scaledObject{&item}})
	}
	return results, nil
}

// scaledObject is a wrapper for keda.sh/v1alpha1.ScaledObject to implement the replicaScaledResource interface
type scaledObject struct {
	*kedav1alpha1.ScaledObject
}

// setReplicas sets the pausedReplicas annotation to the specified replicas. Changes won't be made on Kubernetes until update() is called
func (s *scaledObject) setReplicas(replicas int32) error {
	if replicas == values.Undefined { // pausedAnnotation was not defined before workload was downscaled
		delete(s.Annotations, annotationKedaPausedReplicas)
		return nil
	}
	if s.Annotations == nil {
		s.Annotations = map[string]string{}
	}
	s.Annotations[annotationKedaPausedReplicas] = strconv.Itoa(int(replicas))
	return nil
}

// getReplicas gets the current value of the pausedReplicas annotation
func (s *scaledObject) getReplicas() (int32, error) {
	pausedReplicasAnnotation, ok := s.Annotations[annotationKedaPausedReplicas]
	if !ok {
		return values.Undefined, nil
	}
	pausedReplicas, err := strconv.ParseInt(pausedReplicasAnnotation, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid value for annotation %q: %w", annotationKedaPausedReplicas, err)
	}
	// #nosec G115
	return int32(pausedReplicas), nil
}

// Update updates the resource with all changes made to it. It should only be called once on a resource
func (s *scaledObject) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Keda.KedaV1alpha1().ScaledObjects(s.Namespace).Update(ctx, s.ScaledObject, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update scaledObject: %w", err)
	}

	return nil
}
