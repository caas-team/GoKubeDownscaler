package main

type NamespaceScopeRetrieveError struct {
	Namespace string
}

func newNamespaceScopeRetrieveError(namespace string) error {
	return &NamespaceScopeRetrieveError{Namespace: namespace}
}

func (e *NamespaceScopeRetrieveError) Error() string {
	return "failed to get namespace scope for namespace " + e.Namespace
}
