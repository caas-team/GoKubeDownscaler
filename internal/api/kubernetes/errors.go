package kubernetes

import (
	"fmt"
)

type Error struct {
	Message string
	Value   any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Value)
}

type KubernetesError struct {
	ErrorType string
	Message   string
}

func (k *KubernetesError) Error() string {
	return fmt.Sprintf("%s: %s", k.ErrorType, k.Message)
}

// NewKubernetesError returns an error when a kubernetes operation fails.
func NewKubernetesError(errorType string, msg string) error {
	return &KubernetesError{errorType, msg}
}

type NamespaceError struct {
	ErrorType string
	Message   string
}

func (n *NamespaceError) Error() string {
	return fmt.Sprintf("%s: %s", n.ErrorType, n.Message)
}

// NewNamespaceError returns an error when a namespace operations fails.
func NewNamespaceError(errorType string, msg string) error {
	return &NamespaceError{errorType, msg}
}

type WorkloadError struct {
	ErrorType string
	Message   string
}

func (w *WorkloadError) Error() string {
	return fmt.Sprintf("%s: %s", w.ErrorType, w.Message)
}

// NewWorkloadError returns an error when a workload operation fails.
func NewWorkloadError(errorType string, msg string) error {
	return &WorkloadError{errorType, msg}
}

type EventError struct {
	ErrorType string
	Message   string
}

func (e *EventError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorType, e.Message)
}

// NewEventError returns an error when an event operation fails.
func NewEventError(errorType string, msg string) error {
	return &WorkloadError{errorType, msg}
}
