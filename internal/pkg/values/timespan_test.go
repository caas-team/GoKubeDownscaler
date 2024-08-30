package values

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var zeroTime = time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestParseRelativeTimeSpan(t *testing.T) {
	tests := []struct {
		name           string
		timespanString string
		wantResult     *relativeTimeSpan
		wantErr        bool
	}{
		{
			name:           "valid",
			timespanString: "Mon-Fri 07:00-16:00 UTC",
			wantResult: &relativeTimeSpan{
				timezone:    time.UTC,
				weekdayFrom: time.Monday,
				weekdayTo:   time.Friday,
				timeFrom:    zeroTime.Add(7 * time.Hour),
				timeTo:      zeroTime.Add(16 * time.Hour),
			},
			wantErr: false,
		},
		{
			name:           "reverse",
			timespanString: "Sat-Sun 20:00-06:00 UTC",
			wantResult: &relativeTimeSpan{
				timezone:    time.UTC,
				weekdayFrom: time.Saturday,
				weekdayTo:   time.Sunday,
				timeFrom:    zeroTime.Add(20 * time.Hour),
				timeTo:      zeroTime.Add(6 * time.Hour),
			},
			wantErr: false,
		},
		{
			name:           "invalid TZ",
			timespanString: "Mon-Fri 07:00-16:00 Invalid",
			wantResult:     nil,
			wantErr:        true,
		},
		{
			name:           "invalid Format",
			timespanString: "Mon-Fri 03:00-04-00 UTC",
			wantResult:     nil,
			wantErr:        true,
		},
		{
			name:           "negative Time",
			timespanString: "Mon-Fri -03:00-04:00 UTC",
			wantResult:     nil,
			wantErr:        true,
		},
		{
			name:           "out of range Time",
			timespanString: "Mon-Fri 00:00-26:00 UTC",
			wantResult:     nil,
			wantErr:        true,
		},
		{
			name:           "all day",
			timespanString: "Mon-Fri 00:00-24:00 UTC",
			wantResult: &relativeTimeSpan{
				timezone:    time.UTC,
				weekdayFrom: time.Monday,
				weekdayTo:   time.Friday,
				timeFrom:    zeroTime,
				timeTo:      zeroTime.Add(24 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult, gotErr := parseRelativeTimeSpan(test.timespanString)
			if test.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}

func TestRelativeTimeSpan_isWeekdayInRange(t *testing.T) {
	tests := []struct {
		name       string
		timespan   relativeTimeSpan
		weekday    time.Weekday
		wantResult bool
	}{
		{
			name:       "in range",
			timespan:   relativeTimeSpan{weekdayFrom: time.Monday, weekdayTo: time.Friday},
			weekday:    time.Wednesday,
			wantResult: true,
		},
		{
			name:       "from in range",
			timespan:   relativeTimeSpan{weekdayFrom: time.Monday, weekdayTo: time.Friday},
			weekday:    time.Monday,
			wantResult: true,
		},
		{
			name:       "to in range",
			timespan:   relativeTimeSpan{weekdayFrom: time.Monday, weekdayTo: time.Friday},
			weekday:    time.Friday,
			wantResult: true,
		},
		{
			name:       "reverse in range",
			timespan:   relativeTimeSpan{weekdayFrom: time.Saturday, weekdayTo: time.Sunday},
			weekday:    time.Saturday,
			wantResult: true,
		},
		{
			name:       "reverse out of range",
			timespan:   relativeTimeSpan{weekdayFrom: time.Saturday, weekdayTo: time.Sunday},
			weekday:    time.Monday,
			wantResult: false,
		},
		{
			name:       "out of range",
			timespan:   relativeTimeSpan{weekdayFrom: time.Monday, weekdayTo: time.Friday},
			weekday:    time.Saturday,
			wantResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult := test.timespan.isWeekdayInRange(test.weekday)
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}

func TestRelativeTimeSpan_isTimeOfDayInRange(t *testing.T) {
	tests := []struct {
		name       string
		timespan   relativeTimeSpan
		timeOfDay  time.Time
		wantResult bool
	}{
		{
			name:       "in range",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(6 * time.Hour), timeTo: zeroTime.Add(20 * time.Hour)},
			timeOfDay:  zeroTime.Add(16 * time.Hour),
			wantResult: true,
		},
		{
			name:       "to out of range",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(6 * time.Hour), timeTo: zeroTime.Add(20 * time.Hour)},
			timeOfDay:  zeroTime.Add(20 * time.Hour),
			wantResult: false,
		},
		{
			name:       "reverse in range",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(18 * time.Hour), timeTo: zeroTime.Add(4 * time.Hour)},
			timeOfDay:  zeroTime.Add(3 * time.Hour),
			wantResult: true,
		},
		{
			name:       "reverse to out of range",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(18 * time.Hour), timeTo: zeroTime.Add(4 * time.Hour)},
			timeOfDay:  zeroTime.Add(4 * time.Hour),
			wantResult: false,
		},
		{
			name:       "from in range",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(6 * time.Hour), timeTo: zeroTime.Add(20 * time.Hour)},
			timeOfDay:  zeroTime.Add(6 * time.Hour),
			wantResult: true,
		},
		{
			name:       "reverse from in range",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(18 * time.Hour), timeTo: zeroTime.Add(4 * time.Hour)},
			timeOfDay:  zeroTime.Add(18 * time.Hour),
			wantResult: true,
		},
		{
			name:       "all day",
			timespan:   relativeTimeSpan{timeFrom: zeroTime, timeTo: zeroTime.Add(24 * time.Hour)},
			timeOfDay:  zeroTime.Add(18 * time.Hour),
			wantResult: true,
		},
		{
			name:       "all day overlap to next day",
			timespan:   relativeTimeSpan{timeFrom: zeroTime, timeTo: zeroTime.Add(24 * time.Hour)},
			timeOfDay:  zeroTime.Add(24*time.Hour - time.Nanosecond),
			wantResult: true,
		},
		{
			name:       "all day start of day",
			timespan:   relativeTimeSpan{timeFrom: zeroTime, timeTo: zeroTime.Add(24 * time.Hour)},
			timeOfDay:  zeroTime,
			wantResult: true,
		},
		{
			name:       "24 never",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(24 * time.Hour), timeTo: zeroTime},
			timeOfDay:  zeroTime.Add(18 * time.Hour),
			wantResult: false,
		},
		{
			name:       "24 never overlap to next day",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(24 * time.Hour), timeTo: zeroTime},
			timeOfDay:  zeroTime.Add(24*time.Hour - time.Nanosecond),
			wantResult: false,
		},
		{
			name:       "24 never start of day",
			timespan:   relativeTimeSpan{timeFrom: zeroTime.Add(24 * time.Hour), timeTo: zeroTime},
			timeOfDay:  zeroTime,
			wantResult: false,
		},
		{
			name:       "0 never",
			timespan:   relativeTimeSpan{timeFrom: zeroTime, timeTo: zeroTime},
			timeOfDay:  zeroTime,
			wantResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult := test.timespan.isTimeOfDayInRange(test.timeOfDay)
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}

func TestGetTimeOfDay(t *testing.T) {
	tests := []struct {
		name       string
		time       time.Time
		wantResult time.Time
	}{
		{
			name:       "utc",
			time:       time.Date(2024, time.April, 12, 10, 20, 0, 0, time.UTC),
			wantResult: time.Date(0, time.January, 1, 10, 20, 0, 0, time.UTC),
		},
		{
			name:       "not utc",
			time:       time.Date(2024, time.April, 12, 10, 20, 0, 0, time.FixedZone("UTC+2", 2*int(time.Hour/time.Second))),
			wantResult: time.Date(0, time.January, 1, 10, 20, 0, 0, time.FixedZone("UTC+2", 2*int(time.Hour/time.Second))),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult := getTimeOfDay(test.time)
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}

func TestAbsoluteTimeSpan_isTimeInSpan(t *testing.T) {
	tests := []struct {
		name       string
		timespan   absoluteTimeSpan
		time       time.Time
		wantResult bool
	}{
		{
			name: "in range",
			timespan: absoluteTimeSpan{
				from: time.Date(2024, time.June, 3, 20, 0, 0, 0, time.UTC),
				to:   time.Date(2024, time.June, 10, 6, 0, 0, 0, time.UTC),
			},
			time:       time.Date(2024, time.June, 9, 12, 34, 2, 152, time.UTC),
			wantResult: true,
		},
		{
			name: "out of range",
			timespan: absoluteTimeSpan{
				from: time.Date(2024, time.November, 1, 22, 0, 0, 0, time.UTC),
				to:   time.Date(2024, time.November, 22, 5, 0, 0, 0, time.UTC),
			},
			time:       time.Date(2024, time.December, 5, 2, 30, 0, 0, time.UTC),
			wantResult: false,
		},
		{
			name: "from in range",
			timespan: absoluteTimeSpan{
				from: time.Date(2024, time.November, 1, 22, 0, 0, 0, time.UTC),
				to:   time.Date(2024, time.November, 22, 5, 0, 0, 0, time.UTC),
			},
			time:       time.Date(2024, time.November, 1, 22, 0, 0, 0, time.UTC),
			wantResult: true,
		},
		{
			name: "to out of range",
			timespan: absoluteTimeSpan{
				from: time.Date(2024, time.November, 1, 22, 0, 0, 0, time.UTC),
				to:   time.Date(2024, time.November, 22, 5, 0, 0, 0, time.UTC),
			},
			time:       time.Date(2024, time.November, 22, 5, 0, 0, 0, time.UTC),
			wantResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult := test.timespan.isTimeInSpan(test.time)
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}

func TestParseAbsoluteTimeSpan(t *testing.T) {
	time1 := time.Date(2024, time.February, 27, 0, 0, 0, 0, time.UTC)
	time2 := time1.Add(48 * time.Hour)

	tests := []struct {
		name           string
		timespanString string
		wantResult     absoluteTimeSpan
		wantErr        bool
	}{
		{
			name:           "valid no spaces",
			timespanString: fmt.Sprintf("%s-%s", time1.Format(time.RFC3339), time2.Format(time.RFC3339)),
			wantResult: absoluteTimeSpan{
				from: time1,
				to:   time2,
			},
			wantErr: false,
		},
		{
			name:           "valid with spaces",
			timespanString: fmt.Sprintf("%s - %s", time1.Format(time.RFC3339), time2.Format(time.RFC3339)),
			wantResult: absoluteTimeSpan{
				from: time1,
				to:   time2,
			},
			wantErr: false,
		},
		{
			name:           "invalid",
			timespanString: "2024-07Z - 2024-07-29T16:00:00+02:00",
			wantResult:     absoluteTimeSpan{},
			wantErr:        true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult, gotErr := parseAbsoluteTimeSpan(test.timespanString)
			if test.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}

func TestIsAbsoluteTimestamp(t *testing.T) {
	time1 := time.Date(2024, time.February, 27, 0, 0, 0, 0, time.UTC)
	time2 := time1.Add(48 * time.Hour)

	tests := []struct {
		name           string
		timespanString string
		wantResult     bool
	}{
		{
			name:           "absolute timespan no spaces",
			timespanString: fmt.Sprintf("%s-%s", time1.Format(time.RFC3339), time2.Format(time.RFC3339)),
			wantResult:     true,
		},
		{
			name:           "absolute timespan with spaces",
			timespanString: fmt.Sprintf("%s - %s", time1.Format(time.RFC3339), time2.Format(time.RFC3339)),
			wantResult:     true,
		},
		{
			name:           "relative timespan",
			timespanString: "Mon-Fri 07:30-20:30 Europe/Berlin",
			wantResult:     false,
		},
		{
			name:           "not a timespan",
			timespanString: "09:00-16:00",
			wantResult:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotResult := isAbsoluteTimestamp(test.timespanString)
			assert.Equal(t, test.wantResult, gotResult)
		})
	}
}
