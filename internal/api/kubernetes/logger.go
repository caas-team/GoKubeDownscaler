package kubernetes

import (
	"context"
	"log/slog"

	"github.com/caas-team/gokubedownscaler/internal/pkg/scalable"
	v1 "k8s.io/api/core/v1"
)

const reasonInvalidConfiguration = "InvalidConfiguration"

// Logger handles logging for both namespaces and workloads.
type ResourceLogger struct {
	logger resourceLogger
}

// NewResourceLogger creates a logger for workloads.
func NewResourceLoggerForWorkload(client Client, workload scalable.Workload) ResourceLogger {
	return ResourceLogger{
		logger: &workloadLogger{
			client:   client,
			workload: workload,
		},
	}
}

// NewResourceLoggerForNamespace creates a logger for namespaces.
func NewResourceLoggerForNamespace(client Client, namespace string) ResourceLogger {
	return ResourceLogger{
		logger: &namespaceLogger{
			client:    client,
			namespace: namespace,
		},
	}
}

// ErrorInvalidAnnotation adds an annotation error on the target (workload or namespace).
func (r ResourceLogger) ErrorInvalidAnnotation(annotation, message string, ctx context.Context) {
	err := r.logger.log(v1.EventTypeWarning, reasonInvalidConfiguration, annotation, message, ctx)
	if err != nil {
		slog.Error("failed to add error event", "error", err)
	}
}

// ErrorIncompatibleFields adds an incompatible fields error on the target (workload or namespace).
func (r ResourceLogger) ErrorIncompatibleFields(message string, ctx context.Context) {
	err := r.logger.log(v1.EventTypeWarning, reasonInvalidConfiguration, reasonInvalidConfiguration, message, ctx)
	if err != nil {
		slog.Error("failed to add error event", "error", err)
	}
}

// resourceLogger is the interface that all loggers (namespace and workload) implement.
type resourceLogger interface {
	log(eventType, reason, identifier, message string, ctx context.Context) error
}

// namespaceLogger is a concrete implementation of resourceLogger for namespaces.
type namespaceLogger struct {
	client    Client
	namespace string
}

func (n *namespaceLogger) log(eventType, reason, identifier, message string, ctx context.Context) error {
	// Create ObjectReference for Namespace
	involvedObject := v1.ObjectReference{
		Kind:       "Namespace",
		Name:       n.namespace,
		APIVersion: "v1",
	}

	// Call the client to add the event
	return n.client.addEvent(eventType, reason, identifier, message, &involvedObject, ctx)
}

// workloadLogger is a concrete implementation of resourceLogger for workloads.
type workloadLogger struct {
	client   Client
	workload scalable.Workload
}

func (w *workloadLogger) log(eventType, reason, identifier, message string, ctx context.Context) error {
	// Create ObjectReference for Workload
	involvedObject := v1.ObjectReference{
		Kind:       w.workload.GroupVersionKind().Kind,
		Namespace:  w.workload.GetNamespace(),
		Name:       w.workload.GetName(),
		UID:        w.workload.GetUID(),
		APIVersion: w.workload.GroupVersionKind().GroupVersion().String(),
	}

	// Call the client to add the event
	return w.client.addEvent(eventType, reason, identifier, message, &involvedObject, ctx)
}
