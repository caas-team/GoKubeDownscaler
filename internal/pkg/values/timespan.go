package values

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
)

// rfc339Regex is a regex that matches an rfc339 timestamp.
const rfc3339Regex = `(.+Z|.+[+-]\d{2}:\d{2})`

// absoluteTimeSpanRegex matches an absolute timespan. It's groups are the two rfc3339 timestamps.
var absoluteTimeSpanRegex = regexp.MustCompile(fmt.Sprintf(`^%s *- *%s$`, rfc3339Regex, rfc3339Regex))

type TimeSpan interface {
	// isTimeInSpan checks if time is in the timespan or not
	isTimeInSpan(time time.Time) bool
}

type timeSpans []TimeSpan

// inTimeSpans checks if current time is in one of the timespans or not.
func (t *timeSpans) inTimeSpans() bool {
	for _, timespan := range *t {
		if !timespan.isTimeInSpan(time.Now()) {
			continue
		}

		return true
	}

	return false
}

// String implementation for timeSpans.
func (t *timeSpans) String() string {
	if *t != nil {
		return util.UndefinedString
	}

	return fmt.Sprint(*t)
}

func (t *timeSpans) Set(value string) error {
	spans := strings.Split(value, ",")
	timespans := make([]TimeSpan, 0, len(spans))

	for _, timespanText := range spans {
		var timespan TimeSpan
		timespanText = strings.TrimSpace(timespanText)

		timespan, ok := parseBooleanTimeSpan(timespanText)
		if ok {
			timespans = append(timespans, timespan)
			continue
		}

		if isAbsoluteTimespan(timespanText) {
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

// parseAbsoluteTimespans parses an absolute timespan. will panic if timespan is not an absolute timespan.
func parseAbsoluteTimeSpan(timespan string) (absoluteTimeSpan, error) {
	timestamps := absoluteTimeSpanRegex.FindStringSubmatch(timespan)[1:]

	fromTime, err := time.Parse(time.RFC3339, strings.TrimSpace(timestamps[0]))
	if err != nil {
		return absoluteTimeSpan{}, fmt.Errorf("failed to parse rfc3339 timestamp: %w", err)
	}

	toTime, err := time.Parse(time.RFC3339, strings.TrimSpace(timestamps[1]))
	if err != nil {
		return absoluteTimeSpan{}, fmt.Errorf("failed to parse rfc3339 timestamp: %w", err)
	}

	return absoluteTimeSpan{
		from: fromTime,
		to:   toTime,
	}, nil
}

func parseRelativeTimeSpan(timespanString string) (*relativeTimeSpan, error) {
	var err error
	timespan := relativeTimeSpan{}

	parts := strings.Split(timespanString, " ")
	if len(parts) != 3 {
		return nil, newInvalidSyntaxError("relative timespan has more spaces than expected", timespanString)
	}

	weekdaySpan := strings.Split(parts[0], "-")
	if len(weekdaySpan) != 2 {
		return nil, newInvalidSyntaxError("the relative timespans weekday span is not in the expected format (e.g. 'Mon-Fri')", parts[0])
	}

	timeSpan := strings.Split(parts[1], "-")
	if len(timeSpan) != 2 {
		return nil, newInvalidSyntaxError("the relative timespans time window is not in the expected format (e.g. '08:00-20:00')", parts[1])
	}

	timezone := parts[2]

	timespan.timezone, err = time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timezone: %w", err)
	}

	timeFrom, err := parseDayTime(timeSpan[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse time of day from: %w", err)
	}

	timespan.timeFrom = *timeFrom

	timeTo, err := parseDayTime(timeSpan[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse time of day to: %w", err)
	}

	timespan.timeTo = *timeTo

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
	timeFrom    dayTime
	timeTo      dayTime
}

// isWeekdayInRange checks if the weekday falls into the weekday range.
func (t relativeTimeSpan) isWeekdayInRange(weekday time.Weekday) bool {
	if t.weekdayFrom <= t.weekdayTo { // check if range wraps across weeks
		return weekday >= t.weekdayFrom && weekday <= t.weekdayTo
	}

	return weekday >= t.weekdayFrom || weekday <= t.weekdayTo
}

// isTimeOfDayInRange checks if the time falls into the time of day range.
func (t relativeTimeSpan) isTimeOfDayInRange(timeOfDay dayTime) bool {
	if t.timeFrom > t.timeTo { // check if range wraps across days
		return timeOfDay >= t.timeFrom || timeOfDay < t.timeTo
	}

	return t.timeFrom <= timeOfDay && t.timeTo > timeOfDay
}

// isTimeInSpan check if the time is in the span.
func (t relativeTimeSpan) isTimeInSpan(targetTime time.Time) bool {
	targetTime = targetTime.In(t.timezone)
	timeOfDay := extractDayTime(targetTime)
	weekday := targetTime.Weekday()

	return t.isTimeOfDayInRange(timeOfDay) && t.isWeekdayInRange(weekday)
}

// String implementation for relativeTimeSpan.
func (t relativeTimeSpan) String() string {
	return fmt.Sprintf(
		"relativeTimeSpan(%.3s-%.3s %s-%s %s)",
		t.weekdayFrom,
		t.weekdayTo,
		t.timeFrom,
		t.timeTo,
		t.timezone,
	)
}

type absoluteTimeSpan struct {
	from time.Time
	to   time.Time
}

// isTimeInSpan check if the time is in the span.
func (t absoluteTimeSpan) isTimeInSpan(targetTime time.Time) bool {
	return (t.from.Before(targetTime) || t.from.Equal(targetTime)) && t.to.After(targetTime)
}

// String implementation for absoluteTimeSpan.
func (t absoluteTimeSpan) String() string {
	return fmt.Sprintf(
		"absoluteTimeSpan(%s - %s)",
		t.from.Format(time.RFC3339),
		t.to.Format(time.RFC3339),
	)
}

// isAbsoluteTimespan checks if the timespan string is of an absolute Timespan.
func isAbsoluteTimespan(timestamp string) bool {
	return absoluteTimeSpanRegex.MatchString(timestamp)
}

// getWeekday gets the weekday from the given string.
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

	return 0, newInvalidSyntaxError(
		"weekday is not in the expected format (e.g. 'mon, tue, wed, thu, fri, sat, sun')",
		weekday,
	)
}

// booleanTimeSpan is a TimeSpan which statically is either always active or never active.
type booleanTimeSpan bool

func (b booleanTimeSpan) isTimeInSpan(_ time.Time) bool { return bool(b) }

// parseBooleanTimeSpan tries to parse the given timespan string to a booleanTimespan.
func parseBooleanTimeSpan(timespanString string) (booleanTimeSpan, bool) {
	switch strings.ToLower(timespanString) {
	case "always", "true":
		return true, true
	case "never", "false":
		return false, true
	}

	return false, false
}
