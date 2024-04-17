package models

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"time"

	"github.com/oklog/ulid/v2"
)

// ULIDGenerator implements models.IDGenerator and generates ULIDs for tokenIDs
type ULIDGenerator struct{}

func (ULIDGenerator) ID() (string, error) {
	now := time.Now()
	ms := ulid.Timestamp(now)
	id, err := ulid.New(ms, rand.Reader)
	if err != nil {
		return "", err
	}
	return id.String(), err
}

// RandomGenerator implements models.IDGenerator and generates random IDs used for session IDs
type RandomGenerator struct {
	Length int
}

func (r RandomGenerator) ID() (string, error) {
	b := make([]byte, r.Length)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func NewRandomGenerator(length int) RandomGenerator {
	return RandomGenerator{length}
}
