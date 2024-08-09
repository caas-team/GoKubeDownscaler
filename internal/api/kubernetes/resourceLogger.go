package kubernetes

import (
	"context"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
)

func NewResourceLogger(client Client, workload scalable.Workload) resourceLogger {
	logger := resourceLogger{
		workload: workload,
		client:   client,
	}
	return logger
}

type resourceLogger struct {
	workload scalable.Workload
	client   Client
}

func (r resourceLogger) Error(reason, id, message string, ctx context.Context) {
	err := r.client.AddErrorEvent(reason, id, message, r.workload, ctx)
	if err != nil {
		slog.Error("failed to add error event to workload", "workload", r.workload.GetName(), "error", err)
		return
	}
}
