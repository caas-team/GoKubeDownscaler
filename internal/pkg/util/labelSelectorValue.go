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
