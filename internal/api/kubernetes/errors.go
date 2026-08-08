package kubernetes

import "fmt"

// NamespaceScopeError identifies a namespace whose scope could not be retrieved or parsed.
type NamespaceScopeError struct {
	namespace string
	err       error
}

// newNamespaceScopeError creates an error for a namespace scope failure.
func newNamespaceScopeError(namespace string, err error) error {
	return &NamespaceScopeError{
		namespace: namespace,
		err:       err,
	}
}

// Error returns the namespace scope failure message.
func (e *NamespaceScopeError) Error() string {
	return fmt.Sprintf("failed to get namespace scope for namespace %q: %v", e.namespace, e.err)
}

// Unwrap returns the underlying namespace scope error.
func (e *NamespaceScopeError) Unwrap() error {
	return e.err
}

// Namespace returns the namespace whose scope failed.
func (e *NamespaceScopeError) Namespace() string {
	return e.namespace
}
