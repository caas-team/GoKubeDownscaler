package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBooleanScalingValue(t *testing.T) {
	t.Parallel()

	for _, value := range []bool{true, false} {
		got := NewBooleanScalingValue(value)

		assert.Nil(t, got.TimeSpans)
		assert.NotNil(t, got.Bool)
		assert.Equal(t, value, *got.Bool)
		assert.Equal(t, map[bool]string{true: "true", false: "false"}[value], got.String())
	}
}

func TestScalingValueFromTimeSpans(t *testing.T) {
	t.Parallel()

	first := timeSpans{booleanTimeSpan(true)}
	second := timeSpans{booleanTimeSpan(false)}

	tests := []struct {
		name   string
		values []timeSpans
		want   []TimeSpan
	}{
		{
			name: "no values",
			want: []TimeSpan{},
		},
		{
			name:   "one value",
			values: []timeSpans{first},
			want:   []TimeSpan(first),
		},
		{
			name:   "multiple values",
			values: []timeSpans{first, second},
			want:   append([]TimeSpan(first), second...),
		},
		{
			name:   "nil values are skipped",
			values: []timeSpans{nil, first, nil, second},
			want:   append([]TimeSpan(first), second...),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := scalingValueFromTimeSpans(test.values...)

			assert.Equal(t, test.want, got.TimeSpans)
			assert.Nil(t, got.Bool)
		})
	}
}

func TestForceScalingValue(t *testing.T) {
	t.Parallel()

	forceDowntime := timeSpans{booleanTimeSpan(true)}
	forceUptime := timeSpans{booleanTimeSpan(false)}

	tests := []struct {
		name  string
		scope *Scope
		want  []TimeSpan
	}{
		{
			name:  "no force values",
			scope: &Scope{},
			want:  []TimeSpan{},
		},
		{
			name:  "force downtime only",
			scope: &Scope{ForceDowntime: forceDowntime},
			want:  []TimeSpan(forceDowntime),
		},
		{
			name:  "force uptime only",
			scope: &Scope{ForceUptime: forceUptime},
			want:  []TimeSpan(forceUptime),
		},
		{
			name:  "force downtime and uptime",
			scope: &Scope{ForceDowntime: forceDowntime, ForceUptime: forceUptime},
			want:  append([]TimeSpan(forceDowntime), forceUptime...),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := forceScalingValue(test.scope)

			assert.Equal(t, test.want, got.TimeSpans)
			assert.Nil(t, got.Bool)
		})
	}
}

func TestScalingValue(t *testing.T) {
	t.Parallel()

	downTime := timeSpans{booleanTimeSpan(true)}
	upTime := timeSpans{booleanTimeSpan(false)}
	downscalePeriod := timeSpans{booleanTimeSpan(true)}
	upscalePeriod := timeSpans{booleanTimeSpan(false)}

	tests := []struct {
		name  string
		scope *Scope
		want  []TimeSpan
	}{
		{
			name:  "no values",
			scope: &Scope{},
			want:  []TimeSpan{},
		},
		{
			name:  "downtime takes precedence",
			scope: &Scope{DownTime: downTime, UpTime: upTime, DownscalePeriod: downscalePeriod, UpscalePeriod: upscalePeriod},
			want:  []TimeSpan(downTime),
		},
		{
			name:  "uptime takes precedence when downtime is unset",
			scope: &Scope{UpTime: upTime, DownscalePeriod: downscalePeriod, UpscalePeriod: upscalePeriod},
			want:  []TimeSpan(upTime),
		},
		{
			name:  "periods are used when time values are unset",
			scope: &Scope{DownscalePeriod: downscalePeriod, UpscalePeriod: upscalePeriod},
			want:  append([]TimeSpan(downscalePeriod), upscalePeriod...),
		},
		{
			name:  "only downscale period",
			scope: &Scope{DownscalePeriod: downscalePeriod},
			want:  []TimeSpan(downscalePeriod),
		},
		{
			name:  "only upscale period",
			scope: &Scope{UpscalePeriod: upscalePeriod},
			want:  []TimeSpan(upscalePeriod),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := scalingValue(test.scope)

			assert.Equal(t, test.want, got.TimeSpans)
			assert.Nil(t, got.Bool)
		})
	}
}

func TestScalingValueString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "true", NewBooleanScalingValue(true).String())
	assert.Equal(t, "[]", (ScalingValue{}).String())

	first := timeSpans{booleanTimeSpan(true)}
	second := timeSpans{booleanTimeSpan(false)}

	assert.Equal(t, "[true]", scalingValueFromTimeSpans(first).String())
	assert.Equal(t, "[true false]", scalingValueFromTimeSpans(first, second).String())
}
