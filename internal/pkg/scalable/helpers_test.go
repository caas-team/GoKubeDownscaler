package scalable

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func boolAsPointer(value bool) *bool {
	return &value
}

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

func intAsPointer(value int32) *int32 {
	return &value
}

// assertIntPointerEqual checks if two int pointers equal in state, being nil or pointing to the same integer value.
func assertIntPointerEqual(t *testing.T, expected, actual *int32) {
	t.Helper()

	if expected == nil {
		assert.Nil(t, actual)
		return
	}

	if assert.NotNil(t, actual) {
		assert.Equal(t, *expected, *actual)
	}
}
