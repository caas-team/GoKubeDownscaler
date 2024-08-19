package values

import (
	"fmt"
	"strings"
)

// StringList is an alias for []string with a Set funciton for the flag package
type StringList []string

func (s *StringList) Set(text string) error {
	values := strings.Split(text, ",")
	for _, value := range values {
		*s = append(*s, strings.TrimSpace(value))
	}
	return nil
}

func (s *StringList) String() string {
	return fmt.Sprint(*s)
}
