package values

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
)

// rfc339Regex is a regex that matches an rfc339 timestamp.
const (
	rfc3339Regex = `(.+Z|.+[+-]\d{2}:\d{2})`
	weekday      = `(?:mon|tue|wed|thu|fri|sat|sun)`
	timeofday    = `\d{2}:\d{2}`
	timezone     = `[A-Za-z0-9/_+-]+`
)

var (
	// relativeTimeSpanRegex matches a relative timespan, supporting several formats.
	relativeTimeSpanRegex = regexp.MustCompile(
		`(?i)^` +
			`(?:(?P<from_weekday>` + weekday + `)-(?P<to_weekday>` + weekday + `)\s+)?` +
			`(?P<from_time>` + timeofday + `)` +
			`-(?P<to_time>` + timeofday + `)` +
			`(?:\s+(?P<timezone>` + timezone + `))?` +
			`$`,
	)

	// absoluteTimeSpanRegex matches an absolute timespan. It's groups are the two rfc3339 timestamps.
	absoluteTimeSpanRegex = regexp.MustCompile(fmt.Sprintf(`^%s *- *%s$`, rfc3339Regex, rfc3339Regex))
)

type TimeSpan interface {
	// isTimeInSpan checks if time is in the timespan or not
	isTimeInSpan(time time.Time, scopes Scopes) (bool, error)
}

type timeSpans []TimeSpan

// inTimeSpans checks if current time is in one of the timespans or not.
func (t *timeSpans) inTimeSpans(scopes Scopes) (bool, error) {
	for _, timespan := range *t {
		isTimeInSpan, err := timespan.isTimeInSpan(time.Now(), scopes)
		if err != nil {
			return false, fmt.Errorf("failed to check timespan: %w", err)
		}

		if !isTimeInSpan {
			continue
		}

		return true, nil
	}

	return false, nil
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

	*t = timespans

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

type relativeTimeSpan struct {
	timezone    *time.Location
	weekdayFrom *time.Weekday
	weekdayTo   *time.Weekday
	timeFrom    *dayTime
	timeTo      *dayTime
}

// defaultTimeSpan fills the missing values of the relativeTimeSpan with the default values from the scopes.
// If a value is missing and there is no default value for it, an error is returned.
func (t relativeTimeSpan) defaultTimeSpan(scopes Scopes) (relativeTimeSpan, error) {
	defaultedTimeSpan := t

	if t.timezone == nil {
		defaultedTimeSpan.timezone = scopes.GetDefaultTimeSpan()
		if defaultedTimeSpan.timezone == nil {
			return defaultedTimeSpan,
				newUndefinedDefaultError("failed to get default timezone from scopes for relative timespan with missing timezone")
		}
	}

	if t.weekdayFrom == nil {
		defaultedTimeSpan.weekdayFrom = scopes.GetDefaultWeekdayFrom()
		if defaultedTimeSpan.weekdayFrom == nil {
			return defaultedTimeSpan,
				newUndefinedDefaultError("failed to get default weekdayFrom from scopes for relative timespan with missing weekframe")
		}
	}

	if t.weekdayTo == nil {
		defaultedTimeSpan.weekdayTo = scopes.GetDefaultWeekdayTo()
		if defaultedTimeSpan.weekdayTo == nil {
			return defaultedTimeSpan,
				newUndefinedDefaultError("failed to get default weekdayTo from scopes for relative timespan with missing weekframe")
		}
	}

	return defaultedTimeSpan, nil
}

// parseRelativeTimeSpan parses a relative timespan. will panic if timespan is not a relative timespan.
// nolint: cyclop, gocyclo // this function is a bit complex but needed to parse the relative timespan string
func parseRelativeTimeSpan(timespanString string) (*relativeTimeSpan, error) {
	var err error

	match := relativeTimeSpanRegex.FindStringSubmatch(timespanString)
	if match == nil {
		return nil, newInvalidSyntaxError("failed to parse relative timespan from:", timespanString)
	}

	names := relativeTimeSpanRegex.SubexpNames()
	timespan := relativeTimeSpan{}

	for index, name := range names {
		if index == 0 || name == "" {
			continue
		}

		switch name {
		case "from_weekday":
			timespan.weekdayFrom, err = getWeekday(match[index])
			if err != nil {
				return nil, fmt.Errorf("failed to parse 'weekdayFrom': %w", err)
			}
		case "to_weekday":
			timespan.weekdayTo, err = getWeekday(match[index])
			if err != nil {
				return nil, fmt.Errorf("failed to parse 'weekdayTo': %w", err)
			}
		case "from_time":
			timespan.timeFrom, err = parseDayTime(match[index])
			if err != nil {
				return nil, fmt.Errorf("failed to parse time of day from: %w", err)
			}
		case "to_time":
			timespan.timeTo, err = parseDayTime(match[index])
			if err != nil {
				return nil, fmt.Errorf("failed to parse time of day to: %w", err)
			}
		case "timezone":
			if match[index] == "" {
				timespan.timezone = nil
			} else {
				timespan.timezone, err = time.LoadLocation(match[index])
				if err != nil {
					return nil, fmt.Errorf("failed to parse timezone: %w", err)
				}
			}
		}
	}

	return &timespan, nil
}

// isWeekdayInRange checks if the weekday falls into the weekday range.
func (t relativeTimeSpan) isWeekdayInRange(weekday time.Weekday) bool {
	if *t.weekdayFrom <= *t.weekdayTo { // check if range wraps across weeks
		return weekday >= *t.weekdayFrom && weekday <= *t.weekdayTo
	}

	return weekday >= *t.weekdayFrom || weekday <= *t.weekdayTo
}

// isTimeOfDayInRange checks if the time falls into the time of day range.
func (t relativeTimeSpan) isTimeOfDayInRange(timeOfDay dayTime) bool {
	if *t.timeFrom > *t.timeTo { // check if range wraps across days
		return timeOfDay >= *t.timeFrom || timeOfDay < *t.timeTo
	}

	return *t.timeFrom <= timeOfDay && *t.timeTo > timeOfDay
}

// isTimeInSpan check if the time is in the span.
func (t relativeTimeSpan) isTimeInSpan(targetTime time.Time, scopes Scopes) (bool, error) {
	defaultedTimeSpan, err := t.defaultTimeSpan(scopes)
	if err != nil {
		return false, fmt.Errorf("failed to fill missing values of relative timespan with default values: %w", err)
	}

	targetTime = targetTime.In(defaultedTimeSpan.timezone)
	timeOfDay := extractDayTime(targetTime)
	weekday := targetTime.Weekday()

	return defaultedTimeSpan.isTimeOfDayInRange(timeOfDay) && defaultedTimeSpan.isWeekdayInRange(weekday), nil
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
func (t absoluteTimeSpan) isTimeInSpan(targetTime time.Time, _ Scopes) (bool, error) {
	return (t.from.Before(targetTime) || t.from.Equal(targetTime)) && t.to.After(targetTime), nil
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
// Returns nil if the weekday string is empty.
func getWeekday(weekday string) (*time.Weekday, error) {
	if weekday == "" {
		return nil, nil //nolint: nilnil // we want to return nil if the weekday is not set
	}

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
		return &day, nil
	}

	return nil, newInvalidSyntaxError(
		"weekday is not in the expected format (e.g. 'mon, tue, wed, thu, fri, sat, sun')",
		weekday,
	)
}

// booleanTimeSpan is a TimeSpan which statically is either always active or never active.
type booleanTimeSpan bool

func (b booleanTimeSpan) isTimeInSpan(_ time.Time, _ Scopes) (bool, error) { return bool(b), nil }

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
