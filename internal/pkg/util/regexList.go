package util

import (
	"fmt"
	"regexp"
	"strings"
)

type RegexList []*regexp.Regexp

func (r *RegexList) Set(text string) error {
	entries := strings.Split(text, ",")
	*r = make(RegexList, 0, len(entries))

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)

		re, err := regexp.Compile(entry)
		if err != nil {
			return fmt.Errorf("failed to compile stringlist entry as a regex: %w", err)
		}

		*r = append(*r, re)
	}

	return nil
}

func (r *RegexList) String() string {
	return fmt.Sprint(*r)
}

func (r *RegexList) CheckMatchesAny(text string) bool {
	if r == nil {
		return false
	}

	for _, entry := range *r {
		if entry.MatchString(text) {
			return true
		}
	}

	return false
}
