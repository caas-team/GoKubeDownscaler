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

type weekFrameValue struct {
	p **WeekFrame
}

func (w *weekFrameValue) String() string {
	if w.p == nil || *w.p == nil {
		return ""
	}

	weekFrame := *w.p
	if weekFrame.WeekdayFrom == nil || weekFrame.WeekdayTo == nil {
		return ""
	}

	return fmt.Sprintf("%s-%s",
		weekFrame.WeekdayFrom.String()[:3],
		weekFrame.WeekdayTo.String()[:3],
	)
}

func (w *weekFrameValue) Set(value string) error {
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

	if w.p == nil {
		return newNilWeekframe("weekframe pointer is nil")
	}

	if *w.p == nil {
		*w.p = &WeekFrame{}
	}

	(*w.p).WeekdayFrom = &weekdayFrom
	(*w.p).WeekdayTo = &weekdayTo

	return nil
}
