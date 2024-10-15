package models

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestULIDGenerator(t *testing.T) {
	generator := ULIDGenerator{}

	ulid, err := generator.ID()

	require.NoError(t, err)
	assert.NotEmpty(t, ulid)
}

func TestRandomGenerator(t *testing.T) {
	generator := NewRandomGenerator(32)

	id, err := generator.ID()

	require.NoError(t, err)
	assert.NotEmpty(t, id)

	decoded, err := base64.RawURLEncoding.DecodeString(id)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)
}
