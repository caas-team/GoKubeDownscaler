package main

type NamespaceScopeRetrieveError struct {
	namespace string
}

func newNamespaceScopeRetrieveError(namespace string) error {
	return &NamespaceScopeRetrieveError{namespace: namespace}
}

func (n *NamespaceScopeRetrieveError) Error() string {
	return "failed to get namespace scope for namespace " + n.namespace
}
