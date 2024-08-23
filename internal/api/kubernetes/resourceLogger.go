package kubernetes

import (
	"context"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
)

const reasonInvalidConfiguration = "InvalidConfiguration"

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
func (r resourceLogger) ErrorInvalidAnnotation(annotation, message string, ctx context.Context) {
	err := r.client.addErrorEvent(reasonInvalidConfiguration, annotation, message, r.workload, ctx)
	if err != nil {
		slog.Error("failed to add error event to workload", "workload", r.workload.GetName(), "error", err)
		return
	}
}
