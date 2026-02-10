package util

import (
	"fmt"
	"time"
)

type TimezoneValue struct {
	Value *time.Location
}

func (t *TimezoneValue) String() string {
	if t == nil || t.Value == nil {
		return ""
	}

	return t.Value.String()
}

func (t *TimezoneValue) Set(value string) error {
	if t == nil {
		return NewNilTimezoneError("timezone value is nil")
	}

	loc, err := time.LoadLocation(value)
	if err != nil {
		return fmt.Errorf("invalid timezone %q: %w", value, err)
	}

	t.Value = loc

	return nil
}
