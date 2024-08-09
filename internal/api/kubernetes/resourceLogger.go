package kubernetes

import (
	"context"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
)

const reasonInvalidAnnotation = "InvalidAnnotation"

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

// ErrorInvalidAnnotation adds an error on the resource
func (r resourceLogger) ErrorInvalidAnnotation(id, message string, ctx context.Context) {
	err := r.client.AddErrorEvent(reasonInvalidAnnotation, id, message, r.workload, ctx)
	if err != nil {
		slog.Error("failed to add error event to workload", "workload", r.workload.GetName(), "error", err)
		return
	}
}
