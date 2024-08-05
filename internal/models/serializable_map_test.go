package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableMapText(t *testing.T) {
	a := SerializableMap{}
	a["A"] = "1"
	a["B"] = "2"
	data, err := a.MarshalText()
	require.NoError(t, err)
	var b SerializableMap
	err = b.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}
