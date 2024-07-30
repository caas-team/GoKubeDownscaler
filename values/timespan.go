package values

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var errInvalidWeekday = errors.New("error: specified weekday is invalid")

type TimeSpan interface {
	// inTimeSpan checks if time is in the timespan or not
	isTimeInSpan(time.Time) bool
}

type TimeSpans []TimeSpan

// inTimeSpans checks if current time is in one of the timespans or not
func (t *TimeSpans) inTimeSpans() bool {
	for _, timespan := range *t {
		if !timespan.isTimeInSpan(time.Now()) {
			continue
		}
		return true
	}
	return false
}

func (t *TimeSpans) Set(value string) error {
	spans := strings.Split(value, ",")
	var timespans []TimeSpan
	for _, timespanText := range spans {
		if isAbsoluteTimestamp(timespanText) {
			// parse as absolute timestamp
			timespan, err := parseAbsoluteTimeSpan(timespanText)
			if err != nil {
				return fmt.Errorf("failed to parse absolute timespan: %w", err)
			}
			timespans = append(timespans, timespan)
			continue
		}

		// parse as relative timestamp
		timespan, err := parseRelativeTimeSpan(timespanText)
		if err != nil {
			return fmt.Errorf("failed to parse relative timespan: %w", err)
		}
		timespans = append(timespans, timespan)
	}
	*t = TimeSpans(timespans)
	return nil
}

func (t *TimeSpans) String() string {
	return fmt.Sprint(*t)
}

func parseAbsoluteTimeSpan(timespan string) (absoluteTimeSpan, error) {
	timestamps := strings.Split(timespan, " - ")
	fromTime, err := time.Parse(time.RFC3339, timestamps[0])
	if err != nil {
		return absoluteTimeSpan{}, fmt.Errorf("failed to parse rfc3339 timestamp: %w", err)
	}
	toTime, err := time.Parse(time.RFC3339, timestamps[1])
	if err != nil {
		return absoluteTimeSpan{}, fmt.Errorf("failed to parse rfc3339 timestamp: %w", err)
	}

	return absoluteTimeSpan{
		from: fromTime,
		to:   toTime,
	}, nil
}

func parseRelativeTimeSpan(timespanString string) (relativeTimeSpan, error) {
	timespan := relativeTimeSpan{}

	parts := strings.Split(timespanString, " ")
	weekdaySpan := strings.Split(parts[0], "-")
	timeSpan := strings.Split(parts[1], "-")
	timezone := parts[2]

	var err error
	timespan.timezone, err = time.LoadLocation(timezone)
	if err != nil {
		return timespan, fmt.Errorf("failed to parse timezon: %w", err)
	}
	timespan.timeFrom, err = time.ParseInLocation("15:04", timeSpan[0], timespan.timezone)
	if err != nil {
		return timespan, fmt.Errorf("failed to parse time: %w", err)
	}
	timespan.timeTo, err = time.ParseInLocation("15:04", timeSpan[0], timespan.timezone)
	if err != nil {
		return timespan, fmt.Errorf("failed to parse time: %w", err)
	}
	timespan.weekdayFrom, err = getWeekday(weekdaySpan[0])
	if err != nil {
		return timespan, fmt.Errorf("failed to parse weekday: %w", err)
	}
	timespan.weekdayTo, err = getWeekday(weekdaySpan[1])
	if err != nil {
		return timespan, fmt.Errorf("failed to parse weekday: %w", err)
	}

	return timespan, nil
}

type relativeTimeSpan struct {
	timezone    *time.Location
	weekdayFrom time.Weekday
	weekdayTo   time.Weekday
	timeFrom    time.Time
	timeTo      time.Time
}

// isWeekdayInRange checks if the weekday falls into the weekday range
func (t relativeTimeSpan) isWeekdayInRange(weekday time.Weekday) bool {
	if t.weekdayFrom <= t.weekdayTo { // check if range wraps across weeks
		return weekday >= t.weekdayFrom && weekday <= t.weekdayTo
	}
	return weekday >= t.weekdayFrom || weekday <= t.weekdayTo
}

// isTimeOfDayInRange checks if the time falls into the time of day range
func (t relativeTimeSpan) isTimeOfDayInRange(timeOfDay time.Time) bool {
	if t.timeFrom.After(t.timeTo) { // check if range wraps across days
		return timeOfDay.After(t.timeFrom) || timeOfDay.Before(t.timeTo)
	}
	return t.timeFrom.Before(timeOfDay) && t.timeTo.After(timeOfDay)
}

func (t relativeTimeSpan) isTimeInSpan(targetTime time.Time) bool {
	targetTime = targetTime.In(t.timezone)
	timeOfDay := targetTime.Truncate(24 * time.Hour)
	weekday := targetTime.Weekday()
	return t.isTimeOfDayInRange(timeOfDay) && t.isWeekdayInRange(weekday)
}

type absoluteTimeSpan struct {
	from time.Time
	to   time.Time
}

func (t absoluteTimeSpan) isTimeInSpan(targetTime time.Time) bool {
	return t.from.Before(targetTime) && t.to.After(targetTime)
}

func isAbsoluteTimestamp(timestamp string) bool {
	return strings.Contains(timestamp, " - ")
}

func getWeekday(weekday string) (time.Weekday, error) {
	weekdays := map[string]time.Weekday{
		"sun": time.Sunday,
		"mon": time.Monday,
		"tue": time.Tuesday,
		"wed": time.Wednesday,
		"thu": time.Thursday,
		"fri": time.Friday,
		"sat": time.Saturday,
	}

	if day, ok := weekdays[strings.ToLower(weekday)]; ok {
		return day, nil
	}

	return 0, errInvalidWeekday
}
