//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	argocd "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getApplications is the getResourceFunc for Argo Applications.
func getApplications(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	applications, err := clientsets.ArgoCD.ArgoprojV1alpha1().Applications(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	results := make([]Workload, 0, len(applications.Items))
	for i := range applications.Items {
		setGroupVersionKindIfEmpty(&applications.Items[i], argocd.SchemeGroupVersion.WithKind(applicationKind))
		results = append(results, &suspendScaledWorkload{&application{&applications.Items[i]}})
	}

	return results, nil
}

// parseApplicationFromBytes parses the admission review and returns the application.
func parseApplicationFromBytes(rawObject []byte) (Workload, error) {
	var app argocd.Application
	if err := json.Unmarshal(rawObject, &app); err != nil {
		return nil, fmt.Errorf("failed to decode Application: %w", err)
	}

	return &suspendScaledWorkload{&application{&app}}, nil
}

// application is a wrapper for argocd.v1alpha1 to implement the suspendScaledResource interface.
type application struct {
	*argocd.Application
}

// Reget regets the resource from the Kubernetes API.
func (a *application) Reget(clientsets *Clientsets, ctx context.Context) error {
	var err error

	a.Application, err = clientsets.ArgoCD.ArgoprojV1alpha1().Applications(a.Namespace).Get(ctx, a.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	setGroupVersionKindIfEmpty(a.Application, argocd.SchemeGroupVersion.WithKind(applicationKind))

	return nil
}

// LogUpscaleSuccessful logs the selfHeal transition for an upscale.
func (a *application) logUpscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("suspended", "selfHeal", a, summary, dryRun, logger)
}

// LogDownscaleSuccessful logs the selfHeal transition for a downscale.
func (a *application) logDownscaleSuccessful(summary *scalingSummary, dryRun bool, logger *slog.Logger) {
	logWorkloadScalingMessage("unsuspended", "selfHeal", a, summary, dryRun, logger)
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the Application.
func (a *application) getSavedResourcesRequests() *metrics.SavedResources {
	return metrics.NewSavedResources(0, 0)
}

// isSuspended is true when selfHeal is off; Argo treats unset as off.
func (a *application) isSuspended() bool {
	return a.Spec.SyncPolicy == nil || a.Spec.SyncPolicy.Automated == nil ||
		a.Spec.SyncPolicy.Automated.SelfHeal == nil || !*a.Spec.SyncPolicy.Automated.SelfHeal
}

// getSuspend gets the current value of the suspend field on the application and the target downscale state for it.
//
// nolint: nonamedreturns // required to better understand the function
func (a *application) getSuspend() (currentValue, targetDownscaleState values.Replicas) {
	return values.BooleanReplicas(a.isSuspended()), values.BooleanReplicas(true)
}

// setSuspend sets the value of the suspend field on the application.
func (a *application) setSuspend(suspend bool) {
	selfHeal := !suspend

	if a.Spec.SyncPolicy == nil {
		a.Spec.SyncPolicy = &argocd.SyncPolicy{}
	}

	if a.Spec.SyncPolicy.Automated == nil {
		a.Spec.SyncPolicy.Automated = &argocd.SyncPolicyAutomated{}
	}

	a.Spec.SyncPolicy.Automated.SelfHeal = &selfHeal
}

// Update updates the resource with all changes made to it. It should only be called once on a resource.
func (a *application) Update(clientsets *Clientsets, ctx context.Context) error {
	_, err := clientsets.ArgoCD.ArgoprojV1alpha1().Applications(a.Namespace).Update(ctx, a.Application, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	return nil
}

// Copy creates a deep copy of the given Workload, which is expected to be a suspendScaledWorkload wrapping an application.
func (a *application) Copy() (Workload, error) {
	if a.Application == nil {
		return nil, newNilUnderlyingObjectError(a.Kind)
	}

	copied := a.DeepCopy()

	return &suspendScaledWorkload{
		suspendScaledResource: &application{
			Application: copied,
		},
	}, nil
}

// Compare compares two application resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (a *application) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	sswCopy, ok := workloadCopy.(*suspendScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*suspendScaledWorkload)(nil), workloadCopy)
	}

	aCopy, ok := sswCopy.suspendScaledResource.(*application)
	if !ok {
		return nil, newExpectTypeGotTypeError((*application)(nil), sswCopy.suspendScaledResource)
	}

	if a.Application == nil || aCopy.Application == nil {
		return nil, newNilUnderlyingObjectError(a.Kind)
	}

	diff, err := jsondiff.Compare(a.Application, aCopy.Application)
	if err != nil {
		return nil, fmt.Errorf("failed to compare applications: %w", err)
	}

	return diff, nil
}
