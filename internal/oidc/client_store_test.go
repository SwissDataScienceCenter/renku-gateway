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
		client: newMockRelyingParty("https://token.url"),
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

	// handler(c.Response(), req)
	assert.Equal(t, http.StatusOK, rec.Code)
	// assert.Equal(t, "", handler)
}
