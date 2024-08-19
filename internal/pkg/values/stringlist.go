package values

import (
	"fmt"
	"strings"
)

// StringList is an alias for []string with a Set funciton for the flag package
type StringList []string

func (s *StringList) Set(text string) error {
	entries := strings.Split(text, ",")
	var trimmedEntries []string
	for _, entry := range entries {
		trimmedEntries = append(trimmedEntries, strings.TrimSpace(entry))
	}
	*s = trimmedEntries
	return nil
}

func (s *StringList) String() string {
	return fmt.Sprint(*s)
}
