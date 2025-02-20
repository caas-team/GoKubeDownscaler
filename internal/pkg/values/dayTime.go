package values

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	Hour   dayTime = 60
	Minute dayTime = 1
)

var errTimeOfDayOutOfRange = errors.New("error: the specified time of day is out of range")

func extractDayTime(t time.Time) dayTime {
	t = t.UTC()
	return dayTime(t.Hour())*Hour +
		dayTime(t.Minute())*Minute
}

func parseDayTime(s string) (*dayTime, error) {
	var result dayTime
	parts := strings.Split(s, ":")

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse hour of daytime: %w", err)
	}

	if hour < 0 || hour > 24 {
		return nil, errTimeOfDayOutOfRange
	}

	result += dayTime(hour) * Hour

	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse minute of daytime: %w", err)
	}

	if minute < 0 || minute >= 60 {
		return nil, errTimeOfDayOutOfRange
	}

	result += dayTime(minute) * Minute

	return &result, nil
}

// dayTime is a integer representing minutes passed in the day
// this will always be in utc
type dayTime int

func (d dayTime) String() string {
	minute := d % Hour
	hour := d - minute

	return fmt.Sprintf("%s:%s", hour, minute)
}
