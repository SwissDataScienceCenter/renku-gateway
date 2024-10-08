package models

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableStringSliceText(t *testing.T) {
	a := SerializableStringSlice{"v1", "v2"}
	data, err := a.MarshalText()
	require.NoError(t, err)
	var b SerializableStringSlice
	err = b.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSerializableStringSliceBinary(t *testing.T) {
	a := SerializableStringSlice{"v1", "v2"}
	data, err := a.MarshalBinary()
	require.NoError(t, err)
	var b SerializableStringSlice
	err = b.UnmarshalBinary(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSerializableStringSliceJSON(t *testing.T) {
	a := SerializableStringSlice{"v1", "v2"}
	data, err := json.Marshal(a)
	require.NoError(t, err)
	var b SerializableStringSlice
	err = json.Unmarshal(data, &b)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestStringSliceDeepEqualWorksAsExpected(t *testing.T) {
	a := SerializableStringSlice{"1", "2"}
	b := SerializableStringSlice{"1", "2"}
	assert.True(t, reflect.DeepEqual(a, b))
	c := SerializableStringSlice{"2", "1"}
	assert.False(t, reflect.DeepEqual(a, c))
}
