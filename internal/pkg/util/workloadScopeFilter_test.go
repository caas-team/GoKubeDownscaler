package util

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabelSelectorValue_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantString string
	}{
		{name: "empty", input: "", wantString: ""},
		{name: "whitespace only", input: "  ", wantString: ""},
		{name: "equality", input: "downscaler/workload-scope=enabled", wantString: "downscaler/workload-scope=enabled"},
		{name: "inequality", input: "downscaler/workload-scope!=enabled", wantString: "downscaler/workload-scope!=enabled"},
		{
			name:       "set based",
			input:      "kubernetes.io/metadata.name in (kube-system,monitoring)",
			wantString: "kubernetes.io/metadata.name in (kube-system,monitoring)",
		},
		{name: "invalid", input: "foo in (", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var selector LabelSelectorValue

			err := selector.Set(test.input)
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.wantString, selector.String())
		})
	}
}

func TestWorkloadScopeFilter_IsDisabled(t *testing.T) {
	t.Parallel()

	newSelector := func(t *testing.T, text string) LabelSelectorValue {
		t.Helper()

		var selector LabelSelectorValue
		require.NoError(t, selector.Set(text))

		return selector
	}

	tests := []struct {
		name      string
		filter    func(t *testing.T) *WorkloadScopeFilter
		namespace string
		labels    map[string]string
		want      bool
	}{
		{
			name:      "nil filter",
			filter:    func(*testing.T) *WorkloadScopeFilter { return nil },
			namespace: "team-a",
			want:      false,
		},
		{
			name:      "empty filter",
			filter:    func(*testing.T) *WorkloadScopeFilter { return &WorkloadScopeFilter{} },
			namespace: "team-a",
			labels:    map[string]string{"foo": "bar"},
			want:      false,
		},
		{
			name: "regex matches",
			filter: func(*testing.T) *WorkloadScopeFilter {
				return &WorkloadScopeFilter{Namespaces: RegexList{regexp.MustCompile("^team-.*$")}}
			},
			namespace: "team-a",
			want:      true,
		},
		{
			name: "regex does not match",
			filter: func(*testing.T) *WorkloadScopeFilter {
				return &WorkloadScopeFilter{Namespaces: RegexList{regexp.MustCompile("^team-.*$")}}
			},
			namespace: "kube-system",
			want:      false,
		},
		{
			name: "selector matches",
			filter: func(t *testing.T) *WorkloadScopeFilter {
				t.Helper()

				return &WorkloadScopeFilter{NamespaceSelector: newSelector(t, "downscaler/workload-scope!=enabled")}
			},
			namespace: "team-a",
			labels:    map[string]string{},
			want:      true,
		},
		{
			name: "selector does not match",
			filter: func(t *testing.T) *WorkloadScopeFilter {
				t.Helper()

				return &WorkloadScopeFilter{NamespaceSelector: newSelector(t, "downscaler/workload-scope!=enabled")}
			},
			namespace: "kube-system",
			labels:    map[string]string{"downscaler/workload-scope": "enabled"},
			want:      false,
		},
		{
			name: "only regex matches with both set",
			filter: func(t *testing.T) *WorkloadScopeFilter {
				t.Helper()

				return &WorkloadScopeFilter{
					Namespaces:        RegexList{regexp.MustCompile("^team-a$")},
					NamespaceSelector: newSelector(t, "restricted=true"),
				}
			},
			namespace: "team-a",
			labels:    map[string]string{},
			want:      true,
		},
		{
			name: "only selector matches with both set",
			filter: func(t *testing.T) *WorkloadScopeFilter {
				t.Helper()

				return &WorkloadScopeFilter{
					Namespaces:        RegexList{regexp.MustCompile("^team-a$")},
					NamespaceSelector: newSelector(t, "restricted=true"),
				}
			},
			namespace: "team-b",
			labels:    map[string]string{"restricted": "true"},
			want:      true,
		},
		{
			name: "neither matches with both set",
			filter: func(t *testing.T) *WorkloadScopeFilter {
				t.Helper()

				return &WorkloadScopeFilter{
					Namespaces:        RegexList{regexp.MustCompile("^team-a$")},
					NamespaceSelector: newSelector(t, "restricted=true"),
				}
			},
			namespace: "team-b",
			labels:    map[string]string{"restricted": "false"},
			want:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.want, test.filter(t).IsDisabled(test.namespace, test.labels))
		})
	}
}
