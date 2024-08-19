package tokenstore

import (
	"testing"

	"github.com/SwissDataScienceCenter/renku-gateway/internal/models"
)

// Check that TokenStore implements TokenStoreInterface.
// This test would fail to compile otherwise.
func TestTokenStoreImplementsInterface(t *testing.T) {
	ts := TokenStore{}
	_ = models.TokenStoreInterface(&ts)
}
