package values

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/caas-team/gokubedownscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
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
		envDowntimeReplicas,
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

func TestScopeGetScopeFromEnv_ParsesDowntimeReplicas(t *testing.T) {
	tests := []struct {
		name     string
		replicas string
		unset    bool
		want     Replicas
		wantErr  bool
	}{
		{
			name:     "absolute replicas parsed",
			replicas: "2",
			want:     AbsoluteReplicas(2),
		},
		{
			name:     "percentage replicas parsed",
			replicas: "50%",
			want:     PercentageReplicas(50),
		},
		{
			name:     "invalid negative replicas returns error",
			replicas: "-3",
			wantErr:  true,
		},
		{
			name:     "malformed replicas returns error",
			replicas: "3x",
			wantErr:  true,
		},
		{
			name:     "invalid percentage returns error",
			replicas: "150%",
			wantErr:  true,
		},
		{
			name:  "unset leaves DownscaleReplicas nil",
			unset: true,
			want:  nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearScopeEnvVars(t)

			if !test.unset {
				t.Setenv(envDowntimeReplicas, test.replicas)
			}

			scope := NewScope()
			err := scope.GetScopeFromEnv()

			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.want, scope.DownscaleReplicas)
		})
	}
}

func TestScopeGetScopeFromEnv_DowntimeReplicasPrecedence(t *testing.T) {
	tests := []struct {
		name      string
		workload  Replicas
		namespace Replicas
		cli       Replicas
		envValue  string
		want      Replicas
	}{
		{
			name:     "env overrides default",
			envValue: "3",
			want:     AbsoluteReplicas(3),
		},
		{
			name:     "cli overrides env",
			cli:      AbsoluteReplicas(1),
			envValue: "3",
			want:     AbsoluteReplicas(1),
		},
		{
			name:      "namespace overrides env",
			namespace: AbsoluteReplicas(4),
			envValue:  "3",
			want:      AbsoluteReplicas(4),
		},
		{
			name:     "workload overrides env",
			workload: PercentageReplicas(25),
			envValue: "3",
			want:     PercentageReplicas(25),
		},
		{
			name: "default used when env unset",
			want: AbsoluteReplicas(0),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearScopeEnvVars(t)

			if test.envValue != "" {
				t.Setenv(envDowntimeReplicas, test.envValue)
			}

			scopeEnv := NewScope()
			require.NoError(t, scopeEnv.GetScopeFromEnv())

			scopeWorkload, scopePolicy, scopeNamespace, scopeCli := NewScope(), NewScope(), NewScope(), NewScope()
			scopeWorkload.DownscaleReplicas = test.workload
			scopeNamespace.DownscaleReplicas = test.namespace
			scopeCli.DownscaleReplicas = test.cli

			scopes := Scopes{scopeWorkload, scopePolicy, scopeNamespace, scopeCli, scopeEnv, GetDefaultScope()}

			got, err := scopes.GetDownscaleReplicas()
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

type mockLogger struct{}

func (mockLogger) ErrorInvalidAnnotation(string, string, context.Context) {}

func (mockLogger) ErrorIncompatibleFields(string, context.Context) {}

func TestScopes_DownscalePolicyPrecedence(t *testing.T) {
	t.Parallel()

	scopePolicy := NewScope()
	err := scopePolicy.GetScopeFromAnnotations(
		PolicyValuesToAnnotations(&v1alpha1.DownscalePolicyValues{ForceDowntime: "true", DownscaleReplicas: "2"}),
		mockLogger{}, nil, t.Context(),
	)
	require.NoError(t, err)

	scopeCli := NewScope()
	scopeCli.DownscaleReplicas = AbsoluteReplicas(3)

	// the namespace scope is below the DownscalePolicy scope, so none of its values apply
	scopeNamespace := NewScope()
	scopeNamespace.ForceUptime = timeSpans{booleanTimeSpan(true)}
	scopeNamespace.DownscaleReplicas = AbsoluteReplicas(1)

	scopeWorkload := NewScope()

	scopes := Scopes{scopeWorkload, scopePolicy, scopeNamespace, scopeCli, NewScope(), GetDefaultScope()}

	decision := scopes.GetCurrentScaling()
	assert.Equal(t, ScalingDown, decision.Scaling)
	assert.Equal(t, ScopeDownscalePolicy, decision.Scope)

	replicas, err := scopes.GetDownscaleReplicas()
	require.NoError(t, err)
	assert.Equal(t, AbsoluteReplicas(2), replicas)

	// the workload scope is above the DownscalePolicy scope, so its values win
	scopeWorkload.DownscaleReplicas = AbsoluteReplicas(4)

	replicas, err = scopes.GetDownscaleReplicas()
	require.NoError(t, err)
	assert.Equal(t, AbsoluteReplicas(4), replicas)
}

// inactiveWindow is an absolute timespan in the past, so it is never active.
const inactiveWindow = "2000-01-01T00:00:00Z - 2000-01-02T00:00:00Z"

// newPolicyAndNamespaceScopes parses the DownscalePolicy and namespace scopes and returns them in their evaluation order.
func newPolicyAndNamespaceScopes(
	t *testing.T,
	policyValues *v1alpha1.DownscalePolicyValues,
	namespaceAnnotations map[string]string,
) Scopes {
	t.Helper()

	logger := slog.New(slog.DiscardHandler)

	scopePolicy := NewScope()
	err := scopePolicy.GetScopeFromAnnotations(PolicyValuesToAnnotations(policyValues), mockLogger{}, logger, t.Context())
	require.NoError(t, err)

	scopeNamespace := NewScope()
	err = scopeNamespace.GetScopeFromAnnotations(namespaceAnnotations, mockLogger{}, logger, t.Context())
	require.NoError(t, err)

	return Scopes{NewScope(), scopePolicy, scopeNamespace, NewScope(), NewScope(), GetDefaultScope()}
}

func TestScopes_DownscalePolicyExclusionOverNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		policyValues         *v1alpha1.DownscalePolicyValues
		namespaceAnnotations map[string]string
		wantExcluded         bool
		wantScope            ScopeID
	}{
		{
			name:                 "namespace exclude applies when the policy doesn't set exclude",
			policyValues:         &v1alpha1.DownscalePolicyValues{DownscaleReplicas: "1"},
			namespaceAnnotations: map[string]string{annotationExclude: "true"},
			wantExcluded:         true,
			wantScope:            ScopeNamespace,
		},
		{
			name:                 "policy exclude set to false cancels namespace exclude",
			policyValues:         &v1alpha1.DownscalePolicyValues{Exclude: "false"},
			namespaceAnnotations: map[string]string{annotationExclude: "true"},
			wantExcluded:         false,
			wantScope:            ScopeNone,
		},
		{
			name:                 "policy exclude with an inactive window cancels namespace exclude",
			policyValues:         &v1alpha1.DownscalePolicyValues{Exclude: inactiveWindow},
			namespaceAnnotations: map[string]string{annotationExclude: "true"},
			wantExcluded:         false,
			wantScope:            ScopeNone,
		},
		{
			name:                 "namespace exclude-until wins over a policy exclude set to false",
			policyValues:         &v1alpha1.DownscalePolicyValues{Exclude: "false"},
			namespaceAnnotations: map[string]string{annotationExcludeUntil: "2999-01-01T00:00:00Z"},
			wantExcluded:         true,
			wantScope:            ScopeNamespace,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scopes := newPolicyAndNamespaceScopes(t, test.policyValues, test.namespaceAnnotations)

			got := scopes.GetExcludedWithScope(scopes)
			assert.Equal(t, test.wantExcluded, got.Matched)
			assert.Equal(t, test.wantScope, got.Scope)
		})
	}
}

func TestScopes_DownscalePolicyForcedScalingOverNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		policyValues         *v1alpha1.DownscalePolicyValues
		namespaceAnnotations map[string]string
		wantScaling          Scaling
		wantScope            ScopeID
	}{
		{
			name:                 "namespace force-uptime applies when the policy doesn't set forced scaling",
			policyValues:         &v1alpha1.DownscalePolicyValues{DownscaleReplicas: "1"},
			namespaceAnnotations: map[string]string{annotationForceUptime: "true"},
			wantScaling:          ScalingUp,
			wantScope:            ScopeNamespace,
		},
		{
			name:                 "policy force-downtime with an inactive window cancels namespace force-uptime",
			policyValues:         &v1alpha1.DownscalePolicyValues{ForceDowntime: inactiveWindow},
			namespaceAnnotations: map[string]string{annotationForceUptime: "true"},
			wantScaling:          ScalingIgnore,
			wantScope:            ScopeDownscalePolicy,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			scopes := newPolicyAndNamespaceScopes(t, test.policyValues, test.namespaceAnnotations)

			got := scopes.GetCurrentScaling()
			assert.Equal(t, test.wantScaling, got.Scaling)
			assert.Equal(t, test.wantScope, got.Scope)
		})
	}
}
