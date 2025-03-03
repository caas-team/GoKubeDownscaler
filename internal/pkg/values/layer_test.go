package values

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultLayerIncompatible(t *testing.T) {
	t.Parallel()

	layer := GetDefaultLayer()
	err := layer.CheckForIncompatibleFields()
	require.NoError(t, err)
}

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
				ForceUptime:   timeSpans{booleanTimeSpan(true)},
				ForceDowntime: timeSpans{booleanTimeSpan(true)},
			},
			wantErr: true,
		},
		{
			name: "forced up and downtime one false",
			layer: Layer{
				ForceUptime:   timeSpans{booleanTimeSpan(false)},
				ForceDowntime: timeSpans{booleanTimeSpan(true)},
			},
			wantErr: true, // this might be changed in the future
		},
		{
			name: "forced up and downtime false",
			layer: Layer{
				ForceUptime:   timeSpans{booleanTimeSpan(false)},
				ForceDowntime: timeSpans{booleanTimeSpan(false)},
			},
			wantErr: true, // this might be changed in the future
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

func TestLayer_getForcedScaling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		layer       Layer
		wantScaling Scaling
	}{
		{
			name: "forceDowntime",
			layer: Layer{
				ForceDowntime: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "forceUptime",
			layer: Layer{
				ForceUptime: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingUp,
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

			scaling := test.layer.getForcedScaling()
			assert.Equal(t, test.wantScaling, scaling)
		})
	}
}

func TestLayers_GetCurrentScaling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		layers      Layers
		wantScaling Scaling
	}{
		{
			name: "ignore false forcing *time",
			layers: Layers{
				&Layer{},
				&Layer{ForceDowntime: timeSpans{booleanTimeSpan(false)}, UpTime: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
				&Layer{DownTime: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "never stops fallthrough",
			layers: Layers{
				&Layer{},
				&Layer{DownTime: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{DownTime: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "force *time never doesn't stop fallthrough",
			layers: Layers{
				&Layer{},
				&Layer{ForceDowntime: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{DownTime: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "none set",
			layers: Layers{
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			wantScaling: ScalingNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scaling := test.layers.GetCurrentScaling()
			assert.Equal(t, test.wantScaling, scaling)
		})
	}
}

func TestLayers_GetExcluded(t *testing.T) {
	t.Parallel()

	timeUntilTrue := time.Now().AddDate(1, 0, 0)
	timeUntilFalse := time.Now().AddDate(-1, 0, 0)

	tests := []struct {
		name   string
		layers Layers
		want   bool
	}{
		{
			name: "none set",
			layers: Layers{
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			want: false,
		},
		{
			name: "explicit include",
			layers: Layers{
				&Layer{},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(false)}},
			},
			want: true,
		},
		{
			name: "explicit include",
			layers: Layers{
				&Layer{Exclude: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(false)}},
			},
			want: false,
		},
		{
			name: "exclude until true",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilTrue},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			want: true,
		},
		{
			name: "exclude until false",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilFalse},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			want: false,
		},
		{
			name: "exclude until and exclude same layer false",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilFalse, Exclude: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			want: false,
		},
		{
			name: "exclude until and exclude true same layer",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilFalse, Exclude: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			want: true,
		},
		{
			name: "exclude until true and exclude same layer",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilTrue, Exclude: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{},
				&Layer{},
				&Layer{},
			},
			want: true,
		},
		{
			name: "exclude until and exclude true different layers",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilFalse},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Layer{},
				&Layer{},
			},
			want: true,
		},
		{
			name: "exclude until true and exclude different layers",
			layers: Layers{
				&Layer{ExcludeUntil: &timeUntilTrue},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{},
			},
			want: true,
		},
		{
			name: "exclude and exclude until different layers",
			layers: Layers{
				&Layer{},
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Layer{ExcludeUntil: &timeUntilFalse},
				&Layer{},
			},
			want: true,
		},
		{
			name: "exclude and exclude until different layers",
			layers: Layers{
				&Layer{},
				&Layer{Exclude: timeSpans{booleanTimeSpan(false)}},
				&Layer{},
				&Layer{ExcludeUntil: &timeUntilTrue},
				&Layer{},
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := test.layers.GetExcluded()
			assert.Equal(t, test.want, got)
		})
	}
}
