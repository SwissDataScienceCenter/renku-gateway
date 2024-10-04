package sessions

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

type NS struct {
	rand models.RandomGenerator
}

func (ns NS) Nonce() (string, error) {
	return ns.rand.ID()
}

func TestGetSigningKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	pemBlock := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyBlock := string(pem.EncodeToMemory(&pemBlock))

	assert.Equal(t, "", privateKeyBlock)

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	pemBlock = pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyBlock := string(pem.EncodeToMemory(&pemBlock))
	assert.Equal(t, "", publicKeyBlock)

	ns := NS{
		rand: models.RandomGenerator{Length: 8},
	}

	opts := jose.SignerOptions{
		NonceSource: ns,
	}

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, &opts)
	require.NoError(t, err)

	content := "Hello, World!"
	signedPayload, err := signer.Sign([]byte(content))
	require.NoError(t, err)

	payload, err := signedPayload.CompactSerialize()
	require.NoError(t, err)

	assert.Equal(t, "", payload)
}
