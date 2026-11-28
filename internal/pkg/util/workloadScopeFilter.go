package util

// WorkloadScopeFilter decides in which namespaces the workload scope is disabled.
// A namespace is matched if its name matches any of the Namespaces regexes
// or if its labels match the NamespaceSelector.
type WorkloadScopeFilter struct {
	// Namespaces is a list of regexes matched against the namespace name.
	Namespaces RegexList
	// NamespaceSelector is a label selector matched against the namespace labels.
	NamespaceSelector LabelSelectorValue
}

// IsDisabled checks if the workload scope is disabled for the given namespace.
func (f *WorkloadScopeFilter) IsDisabled(namespace string, namespaceLabels map[string]string) bool {
	if f == nil {
		return false
	}

	return f.Namespaces.CheckMatchesAny(namespace) || f.NamespaceSelector.Matches(namespaceLabels)
}
