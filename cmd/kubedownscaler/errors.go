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

type ScalingInvalidError struct {
	message string
}

func newScalingInvalidError(message string) error {
	return &ScalingInvalidError{message: message}
}

func (s *ScalingInvalidError) Error() string {
	return s.message
}
