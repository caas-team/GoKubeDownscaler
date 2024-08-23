package values

import (
	"fmt"
	"strings"
)

// StringList is an alias for []string with a Set funciton for the flag package
type StringList []string

func (s *StringList) Set(text string) error {
	entries := strings.Split(text, ",")
	for _, entry := range entries {
		*s = append(*s, strings.TrimSpace(entry))
	}
	return nil
}

func (s *StringList) String() string {
	return fmt.Sprint(*s)
}
