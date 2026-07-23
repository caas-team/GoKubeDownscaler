package values

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func clearScopeEnvVars(t *testing.T) {
	t.Helper()

	keys := []string{
		envUpscalePeriod,
		envUptime,
		envDownscalePeriod,
		envDowntime,
		envTimezone,
		envWeekFrame,
	}

	for _, key := range keys {
		oldValue, existed := os.LookupEnv(key)
		require.NoError(t, os.Unsetenv(key))

		if existed {
			t.Setenv(key, oldValue)
			continue
		}

		t.Cleanup(func() { require.NoError(t, os.Unsetenv(key)) })
	}
}

func TestScopeGetScopeFromEnv_ParsesDefaultTimezone(t *testing.T) {
	tests := []struct {
		name            string
		timezone        string
		wantNil         bool
		wantErr         bool
		wantTimezoneStr string
	}{
		{
			name:            "valid timezone parsed",
			timezone:        "America/Bogota",
			wantTimezoneStr: "America/Bogota",
		},
		{
			name:     "invalid timezone returns error",
			timezone: "Not/ATimezone",
			wantErr:  true,
		},
		{
			name:     "empty timezone leaves DefaultTimezone nil",
			timezone: "",
			wantNil:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearScopeEnvVars(t)

			if test.timezone != "" {
				t.Setenv(envTimezone, test.timezone)
			}

			scope := NewScope()
			err := scope.GetScopeFromEnv()

			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if test.wantNil {
				require.Nil(t, scope.DefaultTimezone)
				return
			}

			require.NotNil(t, scope.DefaultTimezone)
			require.Equal(t, test.wantTimezoneStr, scope.DefaultTimezone.String())
		})
	}
}

func TestScopeGetScopeFromEnv_DefaultTimezoneAppliesToDowntimeWithoutTimezone(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		downtime string
		wantErr  bool
	}{
		{
			name:     "DEFAULT_TIMEZONE applies to timezone-less downtime",
			timezone: "America/Bogota",
			downtime: "Thu-Fri 12:55-13:00",
		},
		{
			name:     "downtime without timezone and without DEFAULT_TIMEZONE fails",
			downtime: "Thu-Fri 12:55-13:00",
			wantErr:  true,
		},
		{
			name:     "downtime with inline timezone ignores DEFAULT_TIMEZONE",
			timezone: "America/Bogota",
			downtime: "Thu-Fri 12:55-13:00 Europe/Berlin",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearScopeEnvVars(t)

			if test.timezone != "" {
				t.Setenv(envTimezone, test.timezone)
			}

			t.Setenv(envDowntime, test.downtime)

			scopeEnv := NewScope()
			err := scopeEnv.GetScopeFromEnv()
			require.NoError(t, err)

			scopes := Scopes{
				NewScope(),
				NewScope(),
				NewScope(),
				scopeEnv,
				GetDefaultScope(),
			}

			_, err = scopeEnv.DownTime.inTimeSpans(scopes)

			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScopeGetScopeFromEnv_DefaultWeekFrameAppliesToDowntimeWithoutWeekdays(t *testing.T) {
	tests := []struct {
		name      string
		weekframe string
		downtime  string
		wantErr   bool
	}{
		{
			name:      "DEFAULT_WEEKFRAME applies to weekday-less downtime",
			weekframe: "Mon-Fri",
			downtime:  "12:55-13:00",
		},
		{
			name:     "downtime without weekdays and without DEFAULT_WEEKFRAME fails",
			downtime: "12:55-13:00",
			wantErr:  true,
		},
		{
			name:      "downtime with inline weekdays ignores DEFAULT_WEEKFRAME",
			weekframe: "Mon-Fri",
			downtime:  "Thu-Fri 12:55-13:00",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearScopeEnvVars(t)

			if test.weekframe != "" {
				t.Setenv(envWeekFrame, test.weekframe)
			}

			// The relative timespan parser requires a timezone to be present
			t.Setenv(envTimezone, "UTC")

			t.Setenv(envDowntime, test.downtime)

			scopeEnv := NewScope()
			err := scopeEnv.GetScopeFromEnv()
			require.NoError(t, err)

			scopes := Scopes{
				NewScope(),
				NewScope(),
				NewScope(),
				scopeEnv,
				GetDefaultScope(),
			}

			_, err = scopeEnv.DownTime.inTimeSpans(scopes)

			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
