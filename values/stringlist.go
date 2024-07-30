package values

import (
	"fmt"
	"strings"
)

type StringList []string

func (s *StringList) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

func (s *StringList) String() string {
	return fmt.Sprint(*s)
}
