package oidc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler(t *testing.T) {
	client := oidcClient{
		client: mockRelyingParty{isPKCE: false, tokenURL: "https://token.url"},
		id:     "renku",
	}
	store := ClientStore{
		"renku": client,
	}

	handler, err := store.AuthHandler("renku", "abcde-12345")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = echo.WrapHandler(handler)(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("location"), "client_id=mock-client&response_type=code&state=abcde-12345")
}

func TestAuthHandlerInvalidProvider(t *testing.T) {
	client := oidcClient{
		client: mockRelyingParty{isPKCE: false, tokenURL: "https://token.url"},
		id:     "renku",
	}
	store := ClientStore{
		"renku": client,
	}

	_, err := store.AuthHandler("another", "abcde-12345")
	assert.ErrorContains(t, err, "cannot find the provider with ID another")
}
