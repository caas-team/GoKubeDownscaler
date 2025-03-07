package values

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScope_checkForIncompatibleFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scope   Scope
		wantErr bool
	}{
		{
			name: "forced up and downtime",
			scope: Scope{
				ForceUptime:   triStateBool{isSet: true, value: true},
				ForceDowntime: triStateBool{isSet: true, value: true},
			},
			wantErr: true,
		},
		{
			name: "downscale replicas invalid",
			scope: Scope{
				DownscaleReplicas: -12,
			},
			wantErr: true,
		},
		{
			name: "up- and downtime",
			scope: Scope{
				UpTime:   timeSpans{relativeTimeSpan{}},
				DownTime: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "uptime an upscaleperiod",
			scope: Scope{
				UpTime:        timeSpans{relativeTimeSpan{}},
				UpscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "uptime and downscaleperiod",
			scope: Scope{
				UpTime:          timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "downtime and upscaleperiod",
			scope: Scope{
				DownTime:      timeSpans{relativeTimeSpan{}},
				UpscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "downtime and downscaleperiod",
			scope: Scope{
				DownTime:        timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "valid",
			scope: Scope{
				DownTime:        timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.scope.CheckForIncompatibleFields()
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScope_getCurrentScaling(t *testing.T) {
	t.Parallel()
	var (
		inTimeSpan = timeSpans{absoluteTimeSpan{
			from: time.Now().Add(-time.Hour),
			to:   time.Now().Add(time.Hour),
		}}
		outOfTimeSpan = timeSpans{absoluteTimeSpan{
			from: time.Now().Add(-2 * time.Hour),
			to:   time.Now().Add(-time.Hour),
		}}
	)

	tests := []struct {
		name        string
		scope       Scope
		wantScaling Scaling
	}{
		{
			name: "in downtime",
			scope: Scope{
				DownTime: inTimeSpan,
			},
			wantScaling: ScalingDown,
		},
		{
			name: "out of downtime",
			scope: Scope{
				DownTime: outOfTimeSpan,
			},
			wantScaling: ScalingUp,
		},
		{
			name: "in uptime",
			scope: Scope{
				UpTime: inTimeSpan,
			},
			wantScaling: ScalingUp,
		},
		{
			name: "out of uptime",
			scope: Scope{
				UpTime: outOfTimeSpan,
			},
			wantScaling: ScalingDown,
		},
		{
			name: "in downscaleperiod",
			scope: Scope{
				DownscalePeriod: inTimeSpan,
			},
			wantScaling: ScalingDown,
		},
		{
			name: "out of downscaleperiod",
			scope: Scope{
				DownscalePeriod: outOfTimeSpan,
			},
			wantScaling: ScalingIgnore,
		},
		{
			name: "in upscaleperiod",
			scope: Scope{
				UpscalePeriod: inTimeSpan,
			},
			wantScaling: ScalingUp,
		},
		{
			name: "out of upscaleperiod",
			scope: Scope{
				UpscalePeriod: outOfTimeSpan,
			},
			wantScaling: ScalingIgnore,
		},
		{
			name:        "none set",
			scope:       Scope{},
			wantScaling: ScalingNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scaling := test.scope.getCurrentScaling()
			assert.Equal(t, test.wantScaling, scaling)
		})
	}
}
