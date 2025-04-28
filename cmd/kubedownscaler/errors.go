package main

import "fmt"

type NamespaceScopeRetrieveError struct {
	namespace string
}

func newNamespaceScopeRetrieveError(namespace string) error {
	return &NamespaceScopeRetrieveError{namespace: namespace}
}

func (n *NamespaceScopeRetrieveError) Error() string {
	return fmt.Sprintf("failed to get namespace scope for namespace %q", n.namespace)
}

type MaxRetriesExceeded struct {
	maxRetries int
}

func newMaxRetriesExceeded(maxRetries int) error {
	return &MaxRetriesExceeded{maxRetries: maxRetries}
}

func (m *MaxRetriesExceeded) Error() string {
	return fmt.Sprintf("failed to scale resource: number of max retries exceeded (%d) will try again in the next cycle", m.maxRetries)
}
