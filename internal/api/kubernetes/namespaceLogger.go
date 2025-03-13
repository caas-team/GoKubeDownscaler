package kubernetes

import (
	"context"
	"log/slog"

	v1 "k8s.io/api/core/v1"
)

func NewNamespaceLogger(client Client, namespace string) namespaceLogger {
	return namespaceLogger{
		namespace: namespace,
		client:    client,
	}
}

type namespaceLogger struct {
	namespace string
	client    Client
}

// ErrorInvalidAnnotation adds an annotation error on the namespace.
func (n namespaceLogger) ErrorInvalidAnnotation(annotation, message string, ctx context.Context) {
	err := n.client.addNamespaceEvent(v1.EventTypeWarning, reasonInvalidConfiguration, annotation, message, n.namespace, ctx)
	if err != nil {
		slog.Error("failed to add error event to namespace", "namespace", n.namespace, "error", err)
		return
	}
}

// ErrorIncompatibleFields adds an incompatible fields error on the namespace.
func (n namespaceLogger) ErrorIncompatibleFields(message string, ctx context.Context) {
	err := n.client.addNamespaceEvent(v1.EventTypeWarning, reasonInvalidConfiguration, reasonInvalidConfiguration, message, n.namespace, ctx)
	if err != nil {
		slog.Error("failed to add error event to namespace", "namespace", n.namespace, "error", err)
		return
	}
}
