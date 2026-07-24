package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDayTimeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   dayTime
		want string
	}{
		{name: "exact hour", in: 8 * Hour, want: "08:00"},
		{name: "hour and minute", in: 13*Hour + 5*Minute, want: "13:05"},
		{name: "end of day", in: 23*Hour + 59*Minute, want: "23:59"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.in.String())
		})
	}
}
