package values

import (
	"fmt"
	"strings"
)

// StringListValue is an alias for []string with a Set funciton for the flag package
type StringListValue []string

func (s *StringListValue) Set(text string) error {
	entries := strings.Split(text, ",")
	var trimmedEntries []string
	for _, entry := range entries {
		trimmedEntries = append(trimmedEntries, strings.TrimSpace(entry))
	}
	*s = trimmedEntries
	return nil
}

func (s *StringListValue) String() string {
	return fmt.Sprint(*s)
}
