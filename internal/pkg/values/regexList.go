package values

import (
	"fmt"
	"regexp"
	"strings"
)

type RegexList []*regexp.Regexp

func (r *RegexList) Set(text string) error {
	entries := strings.Split(text, ",")
	*r = make(RegexList, len(entries))
	for i, entry := range entries {
		entry = strings.TrimSpace(entry)
		re, err := regexp.Compile(entry)
		if err != nil {
			return fmt.Errorf("failed to compile stringlist entry as a regex: %w", err)
		}
		(*r)[i] = re
	}
	return nil
}

func (r *RegexList) String() string {
	return fmt.Sprint(*r)
}

func (r RegexList) CheckMatchesAny(text string) bool {
	for _, entry := range r {
		if entry.MatchString(text) {
			return true
		}
	}
	return false
}
