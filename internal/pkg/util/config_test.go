package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseConfigEnvVars can't run in parallel, because it uses t.Setenv.
func TestParseConfigEnvVars(t *testing.T) {
	tests := []struct {
		name              string
		env               map[string]string
		wantErr           bool
		wantNamespaces    int
		wantSelector      string
		namespace         string
		namespaceLabels   map[string]string
		wantScopeDisabled bool
	}{
		{
			name:              "nothing set",
			env:               map[string]string{},
			namespace:         "team-a",
			wantScopeDisabled: false,
		},
		{
			name: "both set",
			env: map[string]string{
				"DISABLE_WORKLOAD_SCOPE_NAMESPACES":         "^team-.*$, ^dev$",
				"DISABLE_WORKLOAD_SCOPE_NAMESPACE_SELECTOR": "downscaler/workload-scope!=enabled",
			},
			wantNamespaces:    2,
			wantSelector:      "downscaler/workload-scope!=enabled",
			namespace:         "team-a",
			namespaceLabels:   map[string]string{"downscaler/workload-scope": "enabled"},
			wantScopeDisabled: true,
		},
		{
			name: "selector only",
			env: map[string]string{
				"DISABLE_WORKLOAD_SCOPE_NAMESPACE_SELECTOR": "downscaler/workload-scope!=enabled",
			},
			wantSelector:      "downscaler/workload-scope!=enabled",
			namespace:         "team-a",
			namespaceLabels:   map[string]string{},
			wantScopeDisabled: true,
		},
		{
			name:              "empty namespaces is treated as unset",
			env:               map[string]string{"DISABLE_WORKLOAD_SCOPE_NAMESPACES": ""},
			namespace:         "team-a",
			wantScopeDisabled: false,
		},
		{
			name:              "whitespace only namespaces is treated as unset",
			env:               map[string]string{"DISABLE_WORKLOAD_SCOPE_NAMESPACES": "  "},
			namespace:         "team-a",
			wantScopeDisabled: false,
		},
		{
			name:              "empty selector matches nothing",
			env:               map[string]string{"DISABLE_WORKLOAD_SCOPE_NAMESPACE_SELECTOR": ""},
			namespace:         "team-a",
			namespaceLabels:   map[string]string{},
			wantScopeDisabled: false,
		},
		{
			name:    "invalid namespaces regex",
			env:     map[string]string{"DISABLE_WORKLOAD_SCOPE_NAMESPACES": "team-("},
			wantErr: true,
		},
		{
			name:    "invalid selector",
			env:     map[string]string{"DISABLE_WORKLOAD_SCOPE_NAMESPACE_SELECTOR": "foo in ("},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for key, value := range test.env {
				t.Setenv(key, value)
			}

			config := GetDefaultConfig()

			err := config.ParseConfigEnvVars()
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, config.DisableWorkloadScope.Namespaces, test.wantNamespaces)
			require.Equal(t, test.wantSelector, config.DisableWorkloadScope.NamespaceSelector.String())
			require.Equal(t, test.wantScopeDisabled, config.DisableWorkloadScope.IsDisabled(test.namespace, test.namespaceLabels))
		})
	}
}
