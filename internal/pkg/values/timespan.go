package values

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var errInvalidWeekday = errors.New("error: specified weekday is invalid")
var errRelativeTimespanInvalid = errors.New("error: specified relative timespan is invalid")

type TimeSpan interface {
	// inTimeSpan checks if time is in the timespan or not
	isTimeInSpan(time.Time) bool
}

type timeSpans []TimeSpan

// inTimeSpans checks if current time is in one of the timespans or not
func (t *timeSpans) inTimeSpans() bool {
	for _, timespan := range *t {
		if !timespan.isTimeInSpan(time.Now()) {
			continue
		}
		return true
	}
	return false
}

func (t *timeSpans) Set(value string) error {
	spans := strings.Split(value, ",")
	var timespans []TimeSpan
	for _, timespanText := range spans {
		timespanText = strings.TrimSpace(timespanText)

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
	*t = timeSpans(timespans)
	return nil
}

func (t *timeSpans) String() string {
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

func parseRelativeTimeSpan(timespanString string) (*relativeTimeSpan, error) {
	timespan := relativeTimeSpan{}

	parts := strings.Split(timespanString, " ")
	if len(parts) != 3 {
		return nil, errRelativeTimespanInvalid
	}

	weekdaySpan := strings.Split(parts[0], "-")
	if len(weekdaySpan) != 2 {
		return nil, errRelativeTimespanInvalid
	}
	timeSpan := strings.Split(parts[1], "-")
	if len(timeSpan) != 2 {
		return nil, errRelativeTimespanInvalid
	}
	timezone := parts[2]

	var err error
	timespan.timezone, err = time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timezone: %w", err)
	}
	timespan.timeFrom, err = time.ParseInLocation("15:04", timeSpan[0], timespan.timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'timeFrom': %w", err)
	}
	timespan.timeTo, err = time.ParseInLocation("15:04", timeSpan[1], timespan.timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'timeTo': %w", err)
	}
	timespan.weekdayFrom, err = getWeekday(weekdaySpan[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'weekdayFrom': %w", err)
	}
	timespan.weekdayTo, err = getWeekday(weekdaySpan[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse 'weekdayTo': %w", err)
	}

	return &timespan, nil
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
		return timeOfDay.After(t.timeFrom) || timeOfDay.Equal(t.timeFrom) || timeOfDay.Before(t.timeTo)
	}
	return (t.timeFrom.Before(timeOfDay) || t.timeFrom.Equal(timeOfDay)) && t.timeTo.After(timeOfDay)
}

// isTimeInSpan check if the time is in the span
func (t relativeTimeSpan) isTimeInSpan(targetTime time.Time) bool {
	targetTime = targetTime.In(t.timezone)
	timeOfDay := getTimeOfDay(targetTime)
	weekday := targetTime.Weekday()
	return t.isTimeOfDayInRange(timeOfDay) && t.isWeekdayInRange(weekday)
}

type absoluteTimeSpan struct {
	from time.Time
	to   time.Time
}

// isTimeInSpan check if the time is in the span
func (t absoluteTimeSpan) isTimeInSpan(targetTime time.Time) bool {
	return (t.from.Before(targetTime) || t.from.Equal(targetTime)) && t.to.After(targetTime)
}

// isAbsoluteTimestamp checks if timestamp string is absolute
func isAbsoluteTimestamp(timestamp string) bool {
	return strings.Contains(timestamp, " - ")
}

// getWeekday gets the weekday from the given string
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

// getTimeOfDay gets the time of day of the given time
func getTimeOfDay(targetTime time.Time) time.Time {
	return time.Date(0, time.January, 1,
		targetTime.Hour(),
		targetTime.Minute(),
		targetTime.Second(),
		targetTime.Nanosecond(),
		targetTime.Location(),
	)
}

// overlapsWith function for relativeTimeSpan
func (r relativeTimeSpan) overlapsWith(span TimeSpan) bool {
	switch o := span.(type) {
	case relativeTimeSpan:
		// Check if two timespan are overlapping
		if r.weekdayFrom <= o.weekdayTo && o.weekdayFrom <= r.weekdayTo {
			return r.timeFrom.Before(o.timeTo) && o.timeFrom.Before(r.timeTo)
		}
		return false

	case absoluteTimeSpan:
		// Check if relative overlaps with absolute
		for day := r.weekdayFrom; day <= r.weekdayTo; day++ {
			// Convert the weekday to a specific date within the absolute timespan range
			for testDate := o.from; !testDate.After(o.to); testDate = testDate.AddDate(0, 0, 1) {
				if testDate.Weekday() == day {
					// Create a time value for the relative timespan on this specific date
					startTime := time.Date(testDate.Year(), testDate.Month(), testDate.Day(),
						r.timeFrom.Hour(), r.timeFrom.Minute(), r.timeFrom.Second(), r.timeFrom.Nanosecond(), r.timezone)
					endTime := time.Date(testDate.Year(), testDate.Month(), testDate.Day(),
						r.timeTo.Hour(), r.timeTo.Minute(), r.timeTo.Second(), r.timeTo.Nanosecond(), r.timezone)

					// Check if this time falls within the absolute timespan
					if !(endTime.Before(o.from) || startTime.After(o.to)) {
						return true
					}
				}
			}
		}
		return false
	}
	return false
}

// overlapsWith function for absoluteTimeSpan
func (a absoluteTimeSpan) overlapsWith(span TimeSpan) bool {
	switch o := span.(type) {
	case absoluteTimeSpan:
		return !(a.to.Before(o.from) || o.to.Before(a.from))

	case relativeTimeSpan:
		return o.overlapsWith(a)
	}
	return false
}

// function to check if both timespans overlap
func areTimespanOverlapped(span1, span2 TimeSpan) bool {
	switch s1 := span1.(type) {
	case absoluteTimeSpan:
		return s1.overlapsWith(span2)
	case relativeTimeSpan:
		return s1.overlapsWith(span2)
	default:
		return false
	}
}
