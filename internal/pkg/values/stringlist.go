package values

import (
	"fmt"
	"strings"
)

// StringList is an alias for []string with a Set funciton for the flag package
type StringList []string

func (s *StringList) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

func (s *StringList) String() string {
	return fmt.Sprint(*s)
}
