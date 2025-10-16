//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	admissionv1 "k8s.io/api/admission/v1"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	appsv1 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errMinReplicasBoundsExceeded = errors.New("error: an HPAs minReplicas can only be set to int32 values larger than 1")

// getHorizontalPodAutoscalers is the getResourceFunc for horizontalPodAutoscalers.
func getHorizontalPodAutoscalers(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	hpas, err := clientsets.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get horizontalpodautoscalers: %w", err)
	}

	results := make([]Workload, 0, len(hpas.Items))
	for i := range hpas.Items {
		results = append(results, &replicaScaledWorkload{&horizontalPodAutoscaler{&hpas.Items[i]}})
	}

	return results, nil
}

// parseHorizontalPodAutoscalerFromAdmissionRequest parses the admission review and returns the horizontalPodAutoscaler.
//
//nolint:ireturn //required for interface-based factory
func parseHorizontalPodAutoscalerFromAdmissionRequest(review *admissionv1.AdmissionReview) (Workload, error) {
	var hpa appsv1.HorizontalPodAutoscaler
	if err := json.Unmarshal(review.Request.Object.Raw, &hpa); err != nil {
		return nil, fmt.Errorf("failed to decode horizontalpodautoscaler: %w", err)
	}

	return &replicaScaledWorkload{&horizontalPodAutoscaler{&hpa}}, nil
}

// deepCopyHorizontalPodAutoscaler creates a deep copy of the given Workload,
// which is expected to be a replicaScaledWorkload wrapping a horizontalPodAutoscaler.
//
//nolint:ireturn,varnamelen //required for interface-based workflow
func deepCopyHorizontalPodAutoscaler(w Workload) (Workload, error) {
	rsw, ok := w.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), w)
	}

	hpa, ok := rsw.replicaScaledResource.(*horizontalPodAutoscaler)
	if !ok {
		return nil, newExpectTypeGotTypeError((*horizontalPodAutoscaler)(nil), rsw.replicaScaledResource)
	}

	if hpa.HorizontalPodAutoscaler == nil {
		return nil, newNilUnderlyingObjectError(hpa.Kind)
	}

	copied := hpa.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &horizontalPodAutoscaler{
			HorizontalPodAutoscaler: copied,
		},
	}, nil
}

// compareHorizontalPodAutoscalers compares two horizontalPodAutoscaler resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func compareHorizontalPodAutoscalers(workload, workloadCopy Workload) (jsondiff.Patch, error) {
	rsw, ok := workload.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workload)
	}

	hpa, ok := rsw.replicaScaledResource.(*horizontalPodAutoscaler)
	if !ok {
		return nil, newExpectTypeGotTypeError((*horizontalPodAutoscaler)(nil), rsw.replicaScaledResource)
	}

	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	hpaCopy, ok := rswCopy.replicaScaledResource.(*horizontalPodAutoscaler)
	if !ok {
		return nil, newExpectTypeGotTypeError((*horizontalPodAutoscaler)(nil), rswCopy.replicaScaledResource)
	}

	if hpa.HorizontalPodAutoscaler == nil || hpaCopy.HorizontalPodAutoscaler == nil {
		return nil, newNilUnderlyingObjectError(hpa.Kind)
	}

	diff, err := jsondiff.Compare(hpa.HorizontalPodAutoscaler, hpa.HorizontalPodAutoscaler)
	if err != nil {
		return nil, newFailedToCompareWorkloadsError(hpa.Kind, err)
	}

	return diff, nil
}

// horizontalPodAutoscaler is a wrapper for horizontalpodautoscaler.v2.autoscaling to implement the replicaScaledResource interface.
type horizontalPodAutoscaler struct {
	*appsv1.HorizontalPodAutoscaler
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (h *horizontalPodAutoscaler) setReplicas(replicas int32) error {
	if replicas < 1 {
		return errMinReplicasBoundsExceeded
	}

	h.Spec.MinReplicas = &replicas

	return nil
}

// getReplicas gets the current amount of replicas of the resource.
func (h *horizontalPodAutoscaler) getReplicas() (values.Replicas, error) {
	replicas := h.Spec.MinReplicas
	if replicas == nil {
		return nil, newNoReplicasError(h.Kind, h.Name)
	}

	return values.AbsoluteReplicas(*replicas), nil
}

// Reget regets the resource from the Kubernetes API.
func (h *horizontalPodAutoscaler) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	h.HorizontalPodAutoscaler, err = clientsets.
		Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(h.Namespace).Get(ctx, h.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get horizontalpodautoscaler: %w", err)
	}

	return nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the HorizontalPodAutoscaler.
func (h *horizontalPodAutoscaler) getSavedResourcesRequests(_ int32) *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (h *horizontalPodAutoscaler) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.Kubernetes.AutoscalingV2().HorizontalPodAutoscalers(h.Namespace).Update(
		ctx, h.HorizontalPodAutoscaler,
		metav1.UpdateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to update horizontalpodautoscaler: %w", err)
	}

	return nil
}
