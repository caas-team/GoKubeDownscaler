package scalable

import (
	"log/slog"
	"strings"
)

func logWorkloadScalingMessage(action, attribute string, workload scalableResource, summary ScalingSummary, dryRun bool) {
	resource := workloadKind(workload)
	message := "successfully " + action + " " + resource
	dryRunMessage := "running in dry run mode, would have sent update " + resource + " request to " + action + " " + resource

	if dryRun {
		message = dryRunMessage
	}

	logWorkloadMessage(message, attribute, workload, summary, dryRun)
}

func workloadKind(workload scalableResource) string {
	resource := strings.ToLower(workload.GroupVersionKind().Kind)
	if resource == "" {
		return "workload"
	}

	return resource
}

func logWorkloadMessage(message, attribute string, workload scalableResource, summary ScalingSummary, dryRun bool) {
	args := []any{
		"resource", workload.GroupVersionKind().Kind,
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
