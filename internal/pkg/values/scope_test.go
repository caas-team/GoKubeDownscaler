package values

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultScopeIncompatible(t *testing.T) {
	t.Parallel()

	scope := GetDefaultScope()
	err := scope.CheckForIncompatibleFields()
	require.NoError(t, err)
}

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
				ForceUptime:   timeSpans{booleanTimeSpan(true)},
				ForceDowntime: timeSpans{booleanTimeSpan(true)},
			},
			wantErr: true,
		},
		{
			name: "forced up and downtime one false",
			scope: Scope{
				ForceUptime:   timeSpans{booleanTimeSpan(false)},
				ForceDowntime: timeSpans{booleanTimeSpan(true)},
			},
			wantErr: true, // this might be changed in the future
		},
		{
			name: "forced up and downtime false",
			scope: Scope{
				ForceUptime:   timeSpans{booleanTimeSpan(false)},
				ForceDowntime: timeSpans{booleanTimeSpan(false)},
			},
			wantErr: true, // this might be changed in the future
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
			name: "uptime an downscaleperiod",
			scope: Scope{
				UpTime:          timeSpans{relativeTimeSpan{}},
				DownscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "downtime an upscaleperiod",
			scope: Scope{
				DownTime:      timeSpans{relativeTimeSpan{}},
				UpscalePeriod: timeSpans{relativeTimeSpan{}},
			},
			wantErr: true,
		},
		{
			name: "downtime an downscaleperiod",
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

	tests := []struct {
		name        string
		scope       Scope
		wantScaling Scaling
	}{
		{
			name: "in downtime",
			scope: Scope{
				DownTime: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "out of downtime",
			scope: Scope{
				DownTime: timeSpans{booleanTimeSpan(false)},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "in uptime",
			scope: Scope{
				UpTime: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "out of uptime",
			scope: Scope{
				UpTime: timeSpans{booleanTimeSpan(false)},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "in downscaleperiod",
			scope: Scope{
				DownscalePeriod: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "out of downscaleperiod",
			scope: Scope{
				DownscalePeriod: timeSpans{booleanTimeSpan(false)},
			},
			wantScaling: ScalingIgnore,
		},
		{
			name: "in upscaleperiod",
			scope: Scope{
				UpscalePeriod: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "out of upscaleperiod",
			scope: Scope{
				UpscalePeriod: timeSpans{booleanTimeSpan(false)},
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

func TestScope_getForceScaling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scope       Scope
		wantScaling Scaling
	}{
		{
			name: "forceDowntime",
			scope: Scope{
				ForceDowntime: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "forceUptime",
			scope: Scope{
				ForceUptime: timeSpans{booleanTimeSpan(true)},
			},
			wantScaling: ScalingUp,
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

			scaling := test.scope.getForceScaling()
			assert.Equal(t, test.wantScaling, scaling)
		})
	}
}

func TestScopes_GetCurrentScaling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		scopes      Scopes
		wantScaling Scaling
	}{
		{
			name: "ignore false forcing *time",
			scopes: Scopes{
				&Scope{},
				&Scope{ForceDowntime: timeSpans{booleanTimeSpan(false)}, UpTime: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
				&Scope{DownTime: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "never stops fallthrough",
			scopes: Scopes{
				&Scope{},
				&Scope{DownTime: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{DownTime: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
			},
			wantScaling: ScalingUp,
		},
		{
			name: "force *time never doesn't stop fallthrough",
			scopes: Scopes{
				&Scope{},
				&Scope{ForceDowntime: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{DownTime: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
			},
			wantScaling: ScalingDown,
		},
		{
			name: "none set",
			scopes: Scopes{
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			wantScaling: ScalingNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scaling := test.scopes.GetCurrentScaling()
			assert.Equal(t, test.wantScaling, scaling)
		})
	}
}

func TestScopes_GetExcluded(t *testing.T) {
	t.Parallel()

	timeUntilTrue := time.Now().AddDate(1, 0, 0)
	timeUntilFalse := time.Now().AddDate(-1, 0, 0)

	tests := []struct {
		name   string
		scopes Scopes
		want   bool
	}{
		{
			name: "none set",
			scopes: Scopes{
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			want: false,
		},
		{
			name: "explicit include",
			scopes: Scopes{
				&Scope{},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(false)}},
			},
			want: true,
		},
		{
			name: "explicit include",
			scopes: Scopes{
				&Scope{Exclude: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(false)}},
			},
			want: false,
		},
		{
			name: "exclude until true",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilTrue},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			want: true,
		},
		{
			name: "exclude until false",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilFalse},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			want: false,
		},
		{
			name: "exclude until and exclude same scope false",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilFalse, Exclude: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			want: false,
		},
		{
			name: "exclude until and exclude true same scope",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilFalse, Exclude: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			want: true,
		},
		{
			name: "exclude until true and exclude same scope",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilTrue, Exclude: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{},
				&Scope{},
				&Scope{},
			},
			want: true,
		},
		{
			name: "exclude until and exclude true different scopes",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilFalse},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Scope{},
				&Scope{},
			},
			want: true,
		},
		{
			name: "exclude until true and exclude different scopes",
			scopes: Scopes{
				&Scope{ExcludeUntil: &timeUntilTrue},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{},
			},
			want: true,
		},
		{
			name: "exclude and exclude until different scopes",
			scopes: Scopes{
				&Scope{},
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(true)}},
				&Scope{ExcludeUntil: &timeUntilFalse},
				&Scope{},
			},
			want: true,
		},
		{
			name: "exclude and exclude until different scopes",
			scopes: Scopes{
				&Scope{},
				&Scope{Exclude: timeSpans{booleanTimeSpan(false)}},
				&Scope{},
				&Scope{ExcludeUntil: &timeUntilTrue},
				&Scope{},
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := test.scopes.GetExcluded()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestGetWorkloadCreationTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		annotation   string
		annotations  map[string]string
		creationTime time.Time
		want         time.Time
		wantErr      bool
	}{
		{
			name:         "use annotation",
			annotation:   "created",
			annotations:  map[string]string{"created": "2025-02-01T00:00:00Z"},
			creationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:         time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			wantErr:      false,
		},
		{
			name:         "default to creationTime",
			annotation:   "created",
			creationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:      false,
		},
		{
			name:         "use creationTime",
			creationTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := getWorkloadCreationTime(test.annotation, test.annotations, test.creationTime, nil, t.Context())
			assert.Equal(t, test.want, got)

			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
