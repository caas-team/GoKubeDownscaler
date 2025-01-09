package values

import (
	"fmt"
	"strings"
)

// StringListValue is an alias for []string with a Set funciton for the flag package
type StringListValue []string

func (s *StringListValue) Set(text string) error {
	entries := strings.Split(text, ",")
	*s = make(StringListValue, len(entries))
	for i, entry := range entries {
		(*s)[i] = strings.TrimSpace(entry)
	}
	return nil
}

func (s *StringListValue) String() string {
	return fmt.Sprint(*s)
}
