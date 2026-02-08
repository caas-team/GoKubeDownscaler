package util

import (
	"fmt"
	"time"
)

type timezoneValue struct {
	p **time.Location
}

func (t *timezoneValue) String() string {
	if t.p == nil || *t.p == nil {
		return ""
	}

	return (*t.p).String()
}

func (t *timezoneValue) Set(v string) error {
	loc, err := time.LoadLocation(v)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", v, err)
	}

	*t.p = loc

	return nil
}
