package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/api/kubernetes"
	"github.com/caas-team/gokubedownscaler/internal/pkg/policy"
	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	"github.com/caas-team/gokubedownscaler/internal/pkg/values"
)

// downscalePolicies holds the DownscalePolicies used during a single scan.
type downscalePolicies struct {
	enabled     bool
	err         error
	byNamespace map[string][]*policy.Policy
	tracker     *policy.Tracker
}

// loadDownscalePolicies gets the DownscalePolicies for the current scan, if the feature is enabled.
// Policies in excluded namespaces are ignored.
func loadDownscalePolicies(client kubernetes.Client, config *runtimeConfiguration, ctx context.Context) *downscalePolicies {
	if !config.DownscalePolicyEnabled {
		return &downscalePolicies{enabled: false}
	}

	policies, err := client.GetDownscalePolicies(config.IncludeNamespaces, ctx)
	if err != nil {
		slog.Error("failed to get downscale policies, skipping all workloads for this scan", "error", err)

		return &downscalePolicies{enabled: true, err: err}
	}

	byNamespace := make(map[string][]*policy.Policy)
	included := make([]*policy.Policy, 0, len(policies))

	for _, downscalePolicy := range policies {
		if config.ExcludeNamespaces.CheckMatchesAny(downscalePolicy.Object.Namespace) {
			continue
		}

		if !downscalePolicy.Valid() {
			slog.Warn(
				"downscale policy is invalid and will not be applied",
				"downscalePolicy", downscalePolicy.Object.Name,
				"namespace", downscalePolicy.Object.Namespace,
				"error", downscalePolicy.Err,
			)
		}

		byNamespace[downscalePolicy.Object.Namespace] = append(byNamespace[downscalePolicy.Object.Namespace], downscalePolicy)
		included = append(included, downscalePolicy)
	}

	slog.Debug("loaded downscale policies", "amount", len(included))

	return &downscalePolicies{
		enabled:     true,
		byNamespace: byNamespace,
		tracker:     policy.NewTracker(included),
	}
}

// scopeFor gets the DownscalePolicy scope of the workload and records which policy was applied to it.
// If no policy applies, an empty scope is returned.
func (d *downscalePolicies) scopeFor(workload scalable.Workload, logger *slog.Logger) (*values.Scope, *slog.Logger, error) {
	if d == nil || !d.enabled {
		return values.NewScope(), logger, nil
	}

	if d.err != nil {
		return nil, logger, fmt.Errorf("failed to get downscale policies: %w", d.err)
	}

	winner, losers := policy.FindApplicable(d.byNamespace[workload.GetNamespace()], workload)
	if winner == nil {
		return values.NewScope(), logger, nil
	}

	d.tracker.Record(winner, losers, workloadResourceKind(workload)+"/"+workload.GetName())

	logger = logger.With("downscalePolicy", winner.Object.Name)

	for _, loser := range losers {
		logger.Debug("downscale policy lost against an older policy", "conflictingPolicy", loser.Object.Name)
	}

	return winner.Scope, logger, nil
}

// updateStatuses writes the status of all policies whose status changed during the scan.
func (d *downscalePolicies) updateStatuses(client kubernetes.Client, ctx context.Context) {
	if d == nil || !d.enabled || d.err != nil {
		return
	}

	for _, update := range d.tracker.PendingUpdates() {
		err := client.UpdateDownscalePolicyStatus(update, ctx)
		if err != nil {
			slog.Error(
				"failed to update downscale policy status",
				"downscalePolicy", update.Policy.Object.Name,
				"namespace", update.Policy.Object.Namespace,
				"error", err,
			)
		}
	}
}
