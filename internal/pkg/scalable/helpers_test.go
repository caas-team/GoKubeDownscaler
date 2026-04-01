package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string { return &s }

func boolAsPointer(value bool) *bool {
	return &value
}

func int32Ptr(value int32) *int32 { return &value }

// assertBoolPointerEqual checks if two bool pointers equal in state, being nil or pointing to true or false.
func assertBoolPointerEqual(t *testing.T, expected, actual *bool) {
	t.Helper()

	if expected == nil {
		assert.Nil(t, actual)
		return
	}

	if assert.NotNil(t, actual) {
		assert.Equal(t, *expected, *actual)
	}
}
