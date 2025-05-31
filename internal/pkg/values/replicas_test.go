package values

import (
	"testing"

	"github.com/caas-team/gokubedownscaler/internal/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestReplicasValue_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantStr   string
		wantType  string
		expectErr bool
	}{
		{
			name:     "valid absolute replicas",
			input:    "5",
			wantStr:  "5",
			wantType: "AbsoluteReplicas",
		},
		{
			name:     "valid percentage replicas",
			input:    "50%",
			wantStr:  "50%",
			wantType: "PercentageReplicas",
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

			switch concreteReplica := (*replicas.Replicas).(type) {
			case AbsoluteReplicas:
				assert.Equal(t, "AbsoluteReplicas", test.wantType)
				assert.Equal(t, test.wantStr, concreteReplica.String())
			case PercentageReplicas:
				assert.Equal(t, "PercentageReplicas", test.wantType)
				assert.Equal(t, test.wantStr, concreteReplica.String())
			default:
				t.Fatalf("unexpected replica type: %T", concreteReplica)
			}
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

func TestReplicasValue_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		replicas Replicas
		expected string
	}{
		{
			name:     "nil replicas",
			replicas: nil,
			expected: util.UndefinedString,
		},
		{
			name:     "absolute replicas",
			replicas: AbsoluteReplicas(5),
			expected: "5",
		},
		{
			name:     "percentage replicas",
			replicas: PercentageReplicas(40),
			expected: "40%",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var replicas *Replicas

			if test.replicas != nil {
				tmp := test.replicas
				replicas = &tmp
			}

			rv := &ReplicasValue{Replicas: replicas}
			assert.Equal(t, test.expected, rv.String())
		})
	}
}
