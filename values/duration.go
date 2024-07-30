package values

import (
	"fmt"
	"strconv"
	"time"
)

type Duration time.Duration

// Set converts the string value into a duration
func (d *Duration) Set(value string) error {
	// try parsing as integer seconds
	seconds, err := strconv.Atoi(value)
	if err == nil {
		*d = Duration(time.Duration(seconds) * time.Second)
		return nil
	}

	// try parsing as duration string
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("failed parsing duration: %w", err)
	}

	*d = Duration(duration)
	return nil
}

func (d *Duration) String() string {
	return fmt.Sprint(*d)
}
