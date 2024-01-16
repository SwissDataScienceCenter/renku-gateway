package db

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type GCMEncryptor struct {
	cipher cipher.AEAD
}

func (g GCMEncryptor) nonce() ([]byte, error) {
	nonce := make([]byte, g.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return []byte{}, err
	}
	return nonce, nil
}

func (g GCMEncryptor) Encrypt(val string) (string, error) {
	nonce, err := g.nonce()
	if err != nil {
		return "", err
	}
	res := g.cipher.Seal(nil, nonce, []byte(val), nil)
	return string(res), nil
}

func (g GCMEncryptor) Decrypt(val string) (string, error) {
	nonce, err := g.nonce()
	if err != nil {
		return "", err
	}
	res, err := g.cipher.Open(nil, nonce, []byte(val), nil)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func NewGCMEncryptor(secret string) (GCMEncryptor, error) {
	block, err := aes.NewCipher([]byte(secret))
	if err != nil {
		return GCMEncryptor{}, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return GCMEncryptor{}, err
	}
	return GCMEncryptor{aesgcm}, nil
}
