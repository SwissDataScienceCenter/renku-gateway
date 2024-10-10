package views

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testHandler struct{}

func (h *testHandler) getLogout(c echo.Context) error {
	return c.Render(http.StatusOK, "logout", nil)
}

func TestRenderer(t *testing.T) {
	e := echo.New()
	tr, err := NewTemplateRenderer()
	require.NoError(t, err)
	tr.Register(e)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := &testHandler{}

	err = h.getLogout(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)
	html := rec.Body.String()
	assert.True(t, len(html) > 0)
	assert.Contains(t, html, "<!DOCTYPE html>")
}
