package kubernetes

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	"github.com/caas-team/gokubedownscaler/internal/pkg/policy"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDownscalePolicies gets and parses all DownscalePolicies in the given namespaces, or in all namespaces if namespaces is nil.
func (c client) GetDownscalePolicies(namespaces []string, ctx context.Context) ([]*policy.Policy, error) {
	if namespaces == nil {
		namespaces = []string{""}
	}

	var policies []*policy.Policy

	for _, namespace := range namespaces {
		var list v1alpha1.DownscalePolicyList

		err := c.clientsets.Client.List(ctx, &list, ctrlclient.InNamespace(namespace))
		if err != nil {
			return nil, fmt.Errorf("failed to list downscale policies in namespace %q: %w", namespace, err)
		}

		for i := range list.Items {
			policies = append(policies, c.parseDownscalePolicy(&list.Items[i], ctx))
		}
	}

	return policies, nil
}

// parseDownscalePolicy parses the scaling values of the policy, reporting invalid values as events on the policy.
func (c client) parseDownscalePolicy(object *v1alpha1.DownscalePolicy, ctx context.Context) *policy.Policy {
	logger := slog.Default().With("downscalePolicy", object.Name, "namespace", object.Namespace)
	annotations := values.PolicyValuesToAnnotations(&object.Spec.DownscalePolicyValues)

	logger.Debug("parsing downscale policy scope", "values", annotations)

	scope := values.NewScope()

	err := scope.GetScopeFromAnnotations(annotations, NewEventLoggerForDownscalePolicy(c, object), logger, ctx)
	if err != nil {
		err = fmt.Errorf("invalid scaling values: %w", err)
	}

	return policy.New(object, scope, err)
}

// UpdateDownscalePolicyStatus writes the status of the policy.
func (c client) UpdateDownscalePolicyStatus(update policy.StatusUpdate, ctx context.Context) error {
	object := update.Policy.Object

	if c.dryRun {
		slog.Info(
			"running in dry run mode, would have updated the downscale policy status",
			"downscalePolicy", object.Name,
			"namespace", object.Namespace,
			"status", fmt.Sprintf("%+v", update.Status),
		)

		return nil
	}

	// replace the whole status, so zero values and emptied lists are written as well
	updated := object.DeepCopy()
	updated.Status = update.Status

	err := c.clientsets.Client.Status().Update(ctx, updated)
	if err != nil {
		return fmt.Errorf("failed to update status of downscale policy %s/%s: %w", object.Namespace, object.Name, err)
	}

	return nil
}
