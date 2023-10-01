package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableSessionTypeText(t *testing.T) {
	a := NewSessionType(0)
	data, err := a.MarshalText()
	require.NoError(t, err)
	var b SessionType
	err = b.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSerializableSessionTypeBinary(t *testing.T) {
	a := NewSessionType(0)
	data, err := a.MarshalBinary()
	require.NoError(t, err)
	var b SessionType
	err = b.UnmarshalBinary(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}
