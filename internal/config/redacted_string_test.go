package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactedString(t *testing.T) {
	originalString := "some-secret-value"

	redactedString := RedactedString(originalString)

	assert.Equal(t, "<redacted-17-chars>", redactedString.String())

	result, err := redactedString.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "<redacted-17-chars>", string(result))

	result, err = redactedString.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "\"<redacted-17-chars>\"", string(result))

	result, err = redactedString.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, "<redacted-17-chars>", string(result))

	object := map[string]any{
		"secret": redactedString,
	}
	result, err = json.Marshal(object)
	require.NoError(t, err)
	assert.Equal(t, "{\"secret\":\"\\u003credacted-17-chars\\u003e\"}", string(result))
}
