package util

import (
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

func TestLabelSelectorValue_Matches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		selector string
		labels   map[string]string
		want     bool
	}{
		{name: "empty selector matches nothing", selector: "", labels: map[string]string{"foo": "bar"}, want: false},
		{name: "equality matches", selector: "foo=bar", labels: map[string]string{"foo": "bar"}, want: true},
		{name: "equality does not match", selector: "foo=bar", labels: map[string]string{"foo": "baz"}, want: false},
		{name: "inequality matches missing label", selector: "foo!=bar", labels: nil, want: true},
		{name: "inequality does not match", selector: "foo!=bar", labels: map[string]string{"foo": "bar"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var selector LabelSelectorValue
			require.NoError(t, selector.Set(test.selector))
			require.Equal(t, test.want, selector.Matches(test.labels))
		})
	}
}

func TestLabelSelectorValue_NilReceiver(t *testing.T) {
	t.Parallel()

	var selector *LabelSelectorValue

	require.Empty(t, selector.String())
	require.False(t, selector.Matches(map[string]string{"foo": "bar"}))
}
