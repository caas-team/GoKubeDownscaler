package util

import (
	"fmt"
	"strings"
	"time"
)

type WeekFrame struct {
	WeekdayFrom *time.Weekday
	WeekdayTo   *time.Weekday
}

type WeekFrameValue struct {
	Value **WeekFrame
}

func (w *WeekFrameValue) String() string {
	if w.Value == nil || *w.Value == nil {
		return ""
	}

	weekFrame := *w.Value
	if weekFrame.WeekdayFrom == nil || weekFrame.WeekdayTo == nil {
		return ""
	}

	return fmt.Sprintf("%s-%s",
		weekFrame.WeekdayFrom.String()[:3],
		weekFrame.WeekdayTo.String()[:3],
	)
}

func (w *WeekFrameValue) Set(value string) error {
	weekdays := map[string]time.Weekday{
		"mon": time.Monday,
		"tue": time.Tuesday,
		"wed": time.Wednesday,
		"thu": time.Thursday,
		"fri": time.Friday,
		"sat": time.Saturday,
		"sun": time.Sunday,
	}

	parts := strings.Split(strings.ToLower(value), "-")
	if len(parts) != 2 {
		return newInvalidWeekFrameValue("invalid weekframe, expected weekdayFrom-weekdayTo", value)
	}

	weekdayFrom, exists := weekdays[parts[0]]
	if !exists {
		return newInvalidWeekFrameValue("invalid weekdayFrom", parts[0])
	}

	weekdayTo, exists := weekdays[parts[1]]
	if !exists {
		return newInvalidWeekFrameValue("invalid weekdayTo", parts[1])
	}

	if w.Value == nil {
		return newNilWeekFrame("weekframe value is nil")
	}

	if *w.Value == nil {
		*w.Value = &WeekFrame{}
	}

	(*w.Value).WeekdayFrom = &weekdayFrom
	(*w.Value).WeekdayTo = &weekdayTo

	return nil
}
