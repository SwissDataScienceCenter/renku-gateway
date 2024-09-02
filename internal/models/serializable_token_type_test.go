package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableTokenTypeText(t *testing.T) {
	a := AccessTokenType
	data, err := a.MarshalText()
	require.NoError(t, err)
	var b OauthTokenType
	err = b.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSerializableTokenTypeBinary(t *testing.T) {
	a := AccessTokenType
	data, err := a.MarshalBinary()
	require.NoError(t, err)
	var b OauthTokenType
	err = b.UnmarshalBinary(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}
