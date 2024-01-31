package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableIntText(t *testing.T) {
	var a SerializableInt = 10
	data, err := a.MarshalText()
	require.NoError(t, err)
	var b SerializableInt
	err = b.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSerializableIntBinary(t *testing.T) {
	var a SerializableInt = 10
	data, err := a.MarshalBinary()
	require.NoError(t, err)
	var b SerializableInt
	err = b.UnmarshalBinary(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}
