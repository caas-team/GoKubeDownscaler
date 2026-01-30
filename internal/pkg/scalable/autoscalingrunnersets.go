//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	actionsv1alpha1 "github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// getAutoscalingRunnerSets is the getResourceFunc for AutoscalingRunnerSets.
func getAutoscalingRunnerSets(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var runnerSets actionsv1alpha1.AutoscalingRunnerSetList

	err := clientsets.Client.List(ctx, &runnerSets, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get autoscalingrunnersets: %w", err)
	}

	results := make([]Workload, 0, len(runnerSets.Items))
	for i := range runnerSets.Items {
		results = append(results, &replicaScaledWorkload{&autoscalingRunnerSet{&runnerSets.Items[i]}})
	}

	return results, nil
}

// parseAutoscalingRunnerSetFromBytes parses the admission review and returns the autoscalingrunnerset wrapped in a Workload.
func parseAutoscalingRunnerSetFromBytes(rawObject []byte) (Workload, error) {
	var ars actionsv1alpha1.AutoscalingRunnerSet
	if err := json.Unmarshal(rawObject, &ars); err != nil {
		return nil, fmt.Errorf("failed to decode autoscalingrunnerset: %w", err)
	}

	return &replicaScaledWorkload{&autoscalingRunnerSet{&ars}}, nil
}

// autoscalingRunnerSet is a wrapper for cronjob.v1.batch to implement the suspendScaledResource interface.
type autoscalingRunnerSet struct {
	*actionsv1alpha1.AutoscalingRunnerSet
}

func (a *autoscalingRunnerSet) Reget(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: a.Namespace, Name: a.Name}, a.AutoscalingRunnerSet)
	if err != nil {
		return fmt.Errorf("failed to get autoscalingrunnerset: %w", err)
	}

	return nil
}

func (a *autoscalingRunnerSet) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, a.AutoscalingRunnerSet)
	if err != nil {
		return fmt.Errorf("failed to update autoscalingrunnerset: %w", err)
	}

	return nil
}

// setReplicas sets the amount of replicas on the resource.
func (a *autoscalingRunnerSet) setReplicas(replicas int32) error {
	intReplicas := int(replicas)
	a.Spec.MinRunners = &intReplicas

	return nil
}

// getReplicas gets the current amount of replicas of the resource.
//
//nolint:gosec //temporary in-place conversion
func (a *autoscalingRunnerSet) getReplicas() (values.Replicas, error) {
	replicas := a.Spec.MinRunners
	if replicas == nil {
		return nil, newNoReplicasError(a.Kind, a.Name)
	}

	return values.AbsoluteReplicas(int32(*replicas)), nil
}

// getSavedResourcesRequests calculates the saved resource requests based on the difference in replicas.
func (a *autoscalingRunnerSet) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	var totalSavedCPU, totalSavedMemory float64

	for i := range a.Spec.Template.Spec.Containers {
		container := &a.Spec.Template.Spec.Containers[i]
		if container.Resources.Requests != nil {
			totalSavedCPU += container.Resources.Requests.Cpu().AsApproximateFloat64()
			totalSavedMemory += container.Resources.Requests.Memory().AsApproximateFloat64()
		}
	}

	totalSavedCPU *= float64(diffReplicas)
	totalSavedMemory *= float64(diffReplicas)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// Copy creates a deep copy of the workload.
func (a *autoscalingRunnerSet) Copy() (Workload, error) {
	if a.AutoscalingRunnerSet == nil {
		return nil, newNilUnderlyingObjectError(a.Kind)
	}

	copied := a.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &autoscalingRunnerSet{
			AutoscalingRunnerSet: copied,
		},
	}, nil
}

// Compare compares the workload with another workload and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //short names are ok for the workflow of this function
func (a *autoscalingRunnerSet) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	depCopy, ok := rswCopy.replicaScaledResource.(*autoscalingRunnerSet)
	if !ok {
		return nil, newExpectTypeGotTypeError((*autoscalingRunnerSet)(nil), rswCopy.replicaScaledResource)
	}

	if a.AutoscalingRunnerSet == nil || depCopy.AutoscalingRunnerSet == nil {
		return nil, newNilUnderlyingObjectError(a.Kind)
	}

	diff, err := jsondiff.Compare(a.AutoscalingRunnerSet, depCopy.AutoscalingRunnerSet)
	if err != nil {
		return nil, fmt.Errorf("failed to compare autoscalingrunnerset: %w", err)
	}

	return diff, nil
}
