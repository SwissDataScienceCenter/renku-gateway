package models

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializableOrderedMapText(t *testing.T) {
	a := NewSerializableOrderedMap()
	a.Set("A", "1")
	data, err := a.MarshalText()
	require.NoError(t, err)
	var b SerializableOrderedMap
	err = b.UnmarshalText(data)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

func TestSerializableOrderedMapDeepEqualWorksAsExpected(t *testing.T) {
	a := NewSerializableOrderedMap()
	a.Set("A", "1")
	a.Set("B", "2")
	b := NewSerializableOrderedMap()
	b.Set("A", "1")
	b.Set("B", "2")
	assert.False(t, a == b)
	assert.True(t, reflect.DeepEqual(a, b))
	c := NewSerializableOrderedMap()
	c.Set("B", "2")
	c.Set("A", "1")
	assert.False(t, reflect.DeepEqual(a, c))
}
