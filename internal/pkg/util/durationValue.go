package util

import (
	"fmt"
	"strconv"
	"time"
)

// DurationValue is an alias for time.DurationValue with a Set function that allows for durations without a unit.
type DurationValue time.Duration

// Set converts the string value into a duration.
func (d *DurationValue) Set(value string) error {
	// try parsing as integer seconds
	seconds, err := strconv.Atoi(value)
	if err == nil {
		*d = DurationValue(time.Duration(seconds) * time.Second)
		return nil
	}

	// try parsing as duration string
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("failed parsing duration: %w", err)
	}

	*d = DurationValue(duration)

	return nil
}

func (d *DurationValue) String() string {
	return time.Duration(*d).String()
}
