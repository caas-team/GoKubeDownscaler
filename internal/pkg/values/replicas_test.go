package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestReplicasValue_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      Replicas
		expectErr bool
	}{
		{
			name:  "valid absolute replicas",
			input: "5",
			want:  AbsoluteReplicas(5),
		},
		{
			name:  "valid percentage replicas",
			input: "50%",
			want:  PercentageReplicas(50),
		},
		{
			name:      "invalid percentage over 100",
			input:     "150%",
			expectErr: true,
		},
		{
			name:      "invalid negative absolute replicas",
			input:     "-3",
			expectErr: true,
		},
		{
			name:      "invalid non numeric",
			input:     "abc",
			expectErr: true,
		},
		{
			name:  "valid boolean true",
			input: "true",
			want:  BooleanReplicas(true),
		},
		{
			name:  "valid boolean false",
			input: "false",
			want:  BooleanReplicas(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var r Replicas
			replicas := &ReplicasValue{Replicas: &r}

			err := replicas.Set(test.input)
			if test.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, *replicas.Replicas)
			assert.Equal(t, test.want, *replicas.Replicas)
		})
	}
}

func TestBooleanReplicas_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input BooleanReplicas
		want  string
	}{
		{name: "true", input: BooleanReplicas(true), want: "true"},
		{name: "false", input: BooleanReplicas(false), want: "false"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, test.input.String())
		})
	}
}

func TestBooleanReplicas_AsBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input BooleanReplicas
		want  bool
	}{
		{name: "true", input: BooleanReplicas(true), want: true},
		{name: "false", input: BooleanReplicas(false), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.input.AsBool()
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestBooleanReplicas_AsInt32_ReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input BooleanReplicas
	}{
		{name: "true", input: BooleanReplicas(true)},
		{name: "false", input: BooleanReplicas(false)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := test.input.AsInt32()
			var invalidReplicaTypeErr *InvalidReplicaTypeError
			assert.ErrorAs(t, err, &invalidReplicaTypeErr)
		})
	}
}

func TestBooleanReplicas_AsIntStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   BooleanReplicas
		wantStr string
	}{
		{name: "true", input: BooleanReplicas(true), wantStr: "true"},
		{name: "false", input: BooleanReplicas(false), wantStr: "false"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := test.input.AsIntStr()
			assert.Equal(t, test.wantStr, got.String())
		})
	}
}

func TestNewReplicasFromIntOrStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *intstr.IntOrString
		expected Replicas
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "int value",
			input:    &intstr.IntOrString{Type: intstr.Int, IntVal: 10},
			expected: AbsoluteReplicas(10),
		},
		{
			name:     "percentage string",
			input:    &intstr.IntOrString{Type: intstr.String, StrVal: "30%"},
			expected: PercentageReplicas(30),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			replica := NewReplicasFromIntOrStr(test.input)
			if test.expected == nil {
				assert.Nil(t, replica)
				return
			}

			require.NotNil(t, replica)
			assert.Equal(t, test.expected.String(), replica.String())
		})
	}
}
