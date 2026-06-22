//nolint:dupl // necessary to handle different workload types separately
package scalable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/caas-team/gokubedownscaler/internal/pkg/metrics"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	"github.com/wI2L/jsondiff"
	acidv1 "github.com/zalando/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// getPostgresqls is the getResourceFunc for Zalando postgres-operator Postgresqls.
func getPostgresqls(namespace string, clientsets *Clientsets, ctx context.Context) ([]Workload, error) {
	var postgresqls acidv1.PostgresqlList

	err := clientsets.Client.List(ctx, &postgresqls, ctrlclient.InNamespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to get postgresqls: %w", err)
	}

	results := make([]Workload, 0, len(postgresqls.Items))
	for i := range postgresqls.Items {
		results = append(results, &replicaScaledWorkload{&postgresql{&postgresqls.Items[i]}})
	}

	return results, nil
}

// parsePostgresqlFromBytes parses the admission review and returns the postgresql wrapped in a Workload.
func parsePostgresqlFromBytes(rawObject []byte) (Workload, error) {
	var pg acidv1.Postgresql
	if err := json.Unmarshal(rawObject, &pg); err != nil {
		return nil, fmt.Errorf("failed to decode postgresql: %w", err)
	}

	return &replicaScaledWorkload{&postgresql{&pg}}, nil
}

// postgresql is a wrapper for postgresql.v1.acid.zalan.do to implement the replicaScaledResource interface.
type postgresql struct {
	*acidv1.Postgresql
}

func (p *postgresql) Reget(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Get(ctx, ctrlclient.ObjectKey{Namespace: p.Namespace, Name: p.Name}, p.Postgresql)
	if err != nil {
		return fmt.Errorf("failed to get postgresql: %w", err)
	}

	return nil
}

// Update writes the spec changes back. It updates the main resource only; the postgresql status is a
// subresource and is left untouched by the apiserver on a main-resource update.
func (p *postgresql) Update(clientsets *Clientsets, ctx context.Context) error {
	err := clientsets.Client.Update(ctx, p.Postgresql)
	if err != nil {
		return fmt.Errorf("failed to update postgresql: %w", err)
	}

	return nil
}

// setReplicas sets the amount of replicas on the resource. Changes won't be made on Kubernetes until update() is called.
func (p *postgresql) setReplicas(replicas int32) error {
	p.Spec.NumberOfInstances = replicas
	return nil
}

// getReplicas gets the current amount of replicas of the resource.
//
// NumberOfInstances is a value int32 (not a pointer), so it can never be nil and there is no
// newNoReplicasError case like statefulsets/stacks have. The error return is kept only to satisfy
// the replicaScaledResource interface and is always nil here.
func (p *postgresql) getReplicas() (values.Replicas, error) {
	return values.AbsoluteReplicas(p.Spec.NumberOfInstances), nil
}

// getSavedResourcesRequests calculates the total saved resources requests when downscaling the Postgresql.
//
// Unlike pod-template based workloads, the Postgresql CR describes the per-instance request once via the
// embedded *Resources block (often nil). Savings are that per-instance request multiplied by the diff in
// instances. Parse failures are treated as zero so resource accounting never breaks a scale operation.
func (p *postgresql) getSavedResourcesRequests(diffReplicas int32) *metrics.SavedResources {
	totalSavedCPU, totalSavedMemory := float64(0), float64(0)

	if p.Spec.Resources != nil {
		requests := p.Spec.ResourceRequests
		totalSavedCPU = parseQuantityAsFloat(requests.CPU)
		totalSavedMemory = parseQuantityAsFloat(requests.Memory)
	}

	totalSavedCPU *= float64(diffReplicas)
	totalSavedMemory *= float64(diffReplicas)

	return metrics.NewSavedResources(totalSavedCPU, totalSavedMemory)
}

// parseQuantityAsFloat parses an optional Kubernetes quantity string into a float64.
// A nil pointer or an unparseable value yields 0 so resource accounting never breaks a scale operation.
func parseQuantityAsFloat(quantity *string) float64 {
	if quantity == nil {
		return 0
	}

	parsed, err := resource.ParseQuantity(*quantity)
	if err != nil {
		return 0
	}

	return parsed.AsApproximateFloat64()
}

// Copy creates a deep copy of the given Workload, which is expected to be a replicaScaledWorkload wrapping a postgresql.
func (p *postgresql) Copy() (Workload, error) {
	if p.Postgresql == nil {
		return nil, newNilUnderlyingObjectError(p.Kind)
	}

	copied := p.DeepCopy()

	return &replicaScaledWorkload{
		replicaScaledResource: &postgresql{
			Postgresql: copied,
		},
	}, nil
}

// Compare compares two postgresql resources and returns the differences as a jsondiff.Patch.
//
//nolint:varnamelen //required for interface-based workflow
func (p *postgresql) Compare(workloadCopy Workload) (jsondiff.Patch, error) {
	rswCopy, ok := workloadCopy.(*replicaScaledWorkload)
	if !ok {
		return nil, newExpectTypeGotTypeError((*replicaScaledWorkload)(nil), workloadCopy)
	}

	pgCopy, ok := rswCopy.replicaScaledResource.(*postgresql)
	if !ok {
		return nil, newExpectTypeGotTypeError((*postgresql)(nil), rswCopy.replicaScaledResource)
	}

	if p.Postgresql == nil || pgCopy.Postgresql == nil {
		return nil, newNilUnderlyingObjectError(p.Kind)
	}

	diff, err := jsondiff.Compare(p.Postgresql, pgCopy.Postgresql)
	if err != nil {
		return nil, fmt.Errorf("failed to compare postgresqls: %w", err)
	}

	return diff, nil
}
