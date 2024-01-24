package db

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	secretKey := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, secretKey)
	require.NoError(t, err)
	enc, err := NewGCMEncryptor(string(secretKey))
	require.NoError(t, err)
	val := "some-secret-value-123"
	valEnc, err := enc.Encrypt(val)
	require.NoError(t, err)
	assert.NotEqual(t, val, valEnc)
	valDec, err := enc.Decrypt(valEnc)
	require.NoError(t, err)
	assert.NotEqual(t, valDec, valEnc)
	assert.Equal(t, val, valDec)
}

