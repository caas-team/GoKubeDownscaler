package kubernetes

import (
	"context"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	v1 "k8s.io/api/core/v1"
)

const reasonInvalidConfiguration = "InvalidConfiguration"

// Logger handles logging for both namespaces and workloads.
type Logger struct {
	client    Client
	workload  scalable.Workload
	namespace string
}

// NewResourceLogger creates a logger for workloads.
func NewResourceLogger(client Client, workload scalable.Workload) Logger {
	return Logger{
		client:   client,
		workload: workload,
	}
}

// NewNamespaceLogger creates a logger for namespaces.
func NewNamespaceLogger(client Client, namespace string) Logger {
	return Logger{
		client:    client,
		namespace: namespace,
	}
}

// ErrorInvalidAnnotation adds an annotation error on the target (workload or namespace).
func (l Logger) ErrorInvalidAnnotation(annotation, message string, ctx context.Context) {
	var err error

	switch {
	case l.workload != nil:
		// Log for workload
		err = l.client.addEvent(v1.EventTypeWarning, reasonInvalidConfiguration, annotation, message, l.workload, "", ctx)
		if err != nil {
			slog.Error("failed to add error event to workload", "workload", l.workload.GetName(), "error", err)
		}
	case l.namespace != "":
		// Log for namespace
		err = l.client.addEvent(v1.EventTypeWarning, reasonInvalidConfiguration, annotation, message, nil, l.namespace, ctx)
		if err != nil {
			slog.Error("failed to add error event to namespace", "namespace", l.namespace, "error", err)
		}
	}
}

// ErrorIncompatibleFields adds an incompatible fields error on the target (workload or namespace).
func (l Logger) ErrorIncompatibleFields(message string, ctx context.Context) {
	var err error

	switch {
	case l.workload != nil:
		// Log for workload
		err = l.client.addEvent(v1.EventTypeWarning, reasonInvalidConfiguration, reasonInvalidConfiguration, message, l.workload, "", ctx)
		if err != nil {
			slog.Error("failed to add error event to workload", "workload", l.workload.GetName(), "error", err)
		}
	case l.namespace != "":
		// Log for namespace
		err = l.client.addEvent(v1.EventTypeWarning, reasonInvalidConfiguration, reasonInvalidConfiguration, message, nil, l.namespace, ctx)
		if err != nil {
			slog.Error("failed to add error event to namespace", "namespace", l.namespace, "error", err)
		}
	}
}
