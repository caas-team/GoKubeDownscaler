package values

import (
	"fmt"
	"regexp"
	"strings"
)

type RegexList []*regexp.Regexp

func (s *RegexList) Set(text string) error {
	entries := strings.Split(text, ",")
	var reList RegexList
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		re, err := regexp.Compile(entry)
		if err != nil {
			return fmt.Errorf("failed to compile stringlist entry as a regex: %w", err)
		}
		reList = append(reList, re)
	}
	*s = reList
	return nil
}

func (s *RegexList) String() string {
	return fmt.Sprint(*s)
}

func (s RegexList) CheckMatchesAny(text string) bool {
	for _, entry := range s {
		if entry.MatchString(text) {
			return true
		}
	}
	return false
}
