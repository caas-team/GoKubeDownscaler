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
