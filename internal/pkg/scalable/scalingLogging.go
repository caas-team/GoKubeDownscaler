package scalable

import (
	"log/slog"
	"strings"
)

func logWorkloadScalingMessage(action, attribute string, workload scalableResource, summary *ScalingSummary, dryRun bool) {
	kind := strings.ToLower(workload.GroupVersionKind().Kind)
	if kind == "" {
		kind = "workload"
	}

	message := "successfully " + action + " " + kind
	dryRunMessage := "running in dry run mode, would have sent update " + kind + " request to " + action + " " + kind

	if dryRun {
		message = dryRunMessage
	}

	logWorkloadMessage(message, attribute, workload, summary, dryRun)
}

func logWorkloadMessage(message, attribute string, workload scalableResource, summary *ScalingSummary, dryRun bool) {
	args := []any{
		"kind", workload.GroupVersionKind().Kind,
		"workload", workload.GetName(),
		"namespace", workload.GetNamespace(),
		"dry run", dryRun,
	}
	if attribute != "" {
		args = append(args, "changed attribute", attribute)
	}

	args = append(args,
		"from", summary.From,
		"to", summary.To,
	)

	slog.Info(message, args...)
}
