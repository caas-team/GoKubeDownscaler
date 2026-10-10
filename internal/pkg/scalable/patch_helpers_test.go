package scalable

import (
	"encoding/json"
	"testing"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertJSONPatchTransforms(t *testing.T, original any, patchData []byte, expected any) {
	t.Helper()

	t.Logf("patchData: %s", patchData)

	if len(patchData) == 0 {
		assert.Equal(t, expected, original)
		return
	}

	originalJSON, err := json.Marshal(original)
	require.NoError(t, err)

	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	patch, err := jsonpatch.DecodePatch(patchData)
	require.NoError(t, err)

	actualJSON, err := patch.Apply(originalJSON)
	require.NoError(t, err)

	var actualObject any
	require.NoError(t, json.Unmarshal(actualJSON, &actualObject))

	var expectedObject any
	require.NoError(t, json.Unmarshal(expectedJSON, &expectedObject))

	assert.Equal(t, expectedObject, actualObject)
}
