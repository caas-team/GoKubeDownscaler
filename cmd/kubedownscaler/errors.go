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
