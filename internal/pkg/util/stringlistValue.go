package util

import (
	"fmt"
	"strings"
)

// StringListValue is an alias for []string with a Set function for the flag package.
type StringListValue []string

func (s *StringListValue) Set(text string) error {
	entries := strings.Split(text, ",")
	*s = make(StringListValue, 0, len(entries))

	for _, entry := range entries {
		*s = append(*s, strings.TrimSpace(entry))
	}

	return nil
}

func (s *StringListValue) String() string {
	return fmt.Sprint(*s)
}
