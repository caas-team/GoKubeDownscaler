package values

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLayer_checkForIncompatibleFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		layer   Layer
		wantErr bool
	}{
		{
			name: "forced up and downtime",
			layer: Layer{
				ForceUptime:   triStateBool{isSet: true, value: true},
				ForceDowntime: triStateBool{isSet: true, value: true},
			},
			wantErr: true,
		},
		{
			name: "downscale replicas invalid",
			layer: Layer{
				DownscaleReplicas: -12,
			},
			wantErr: true,
		},
		{
			name: "up- and downtime",
			layer: Layer{
				UpTime:   timeSpans{relativeTimeSpan{}},
				DownTime: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "uptime an upscaleperiod",
			layer: Layer{
				UpTime:        timeSpans{relativeTimeSpan{}},
				UpscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "uptime an downscaleperiod",
			layer: Layer{
				UpTime:          timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "downtime an upscaleperiod",
			layer: Layer{
				DownTime:      timeSpans{relativeTimeSpan{}},
				UpscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "downtime an downscaleperiod",
			layer: Layer{
				DownTime:        timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "valid",
			layer: Layer{
				DownTime:        timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.layer.CheckForIncompatibleFields()
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLayer_getCurrentScaling(t *testing.T) {
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
		layer       Layer
		wantScaling Scaling
	}{
		{
			name: "in downtime",
			layer: Layer{
				DownTime: inTimeSpan,
			},
			wantScaling: ScalingDown,
		},
		{
			name: "out of downtime",
			layer: Layer{
				DownTime: outOfTimeSpan,
			},
			wantScaling: ScalingUp,
		},
		{
			name: "in uptime",
			layer: Layer{
				UpTime: inTimeSpan,
			},
			wantScaling: ScalingUp,
		},
		{
			name: "out of uptime",
			layer: Layer{
				UpTime: outOfTimeSpan,
			},
			wantScaling: ScalingDown,
		},
		{
			name: "in downscaleperiod",
			layer: Layer{
				DownscalePeriod: inTimeSpan,
			},
			wantScaling: ScalingDown,
		},
		{
			name: "out of downscaleperiod",
			layer: Layer{
				DownscalePeriod: outOfTimeSpan,
			},
			wantScaling: ScalingIgnore,
		},
		{
			name: "in upscaleperiod",
			layer: Layer{
				UpscalePeriod: inTimeSpan,
			},
			wantScaling: ScalingUp,
		},
		{
			name: "out of upscaleperiod",
			layer: Layer{
				UpscalePeriod: outOfTimeSpan,
			},
			wantScaling: ScalingIgnore,
		},
		{
			name:        "none set",
			layer:       Layer{},
			wantScaling: ScalingNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scaling := test.layer.getCurrentScaling()
			assert.Equal(t, test.wantScaling, scaling)
		})
	}
}
