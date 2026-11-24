package util

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
)

// LabelSelectorValue is a label selector with a Set function for the flag package.
// An empty value matches nothing.
type LabelSelectorValue struct {
	selector labels.Selector
}

func (l *LabelSelectorValue) Set(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		l.selector = nil
		return nil
	}

	selector, err := labels.Parse(text)
	if err != nil {
		return fmt.Errorf("failed to parse label selector: %w", err)
	}

	l.selector = selector

	return nil
}

func (l *LabelSelectorValue) String() string {
	if l == nil || l.selector == nil {
		return ""
	}

	return l.selector.String()
}

// Matches checks if the selector is set and matches the given labels.
func (l *LabelSelectorValue) Matches(set map[string]string) bool {
	if l == nil || l.selector == nil {
		return false
	}

	return l.selector.Matches(labels.Set(set))
}

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
