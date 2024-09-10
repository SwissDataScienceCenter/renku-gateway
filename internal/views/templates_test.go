package views

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTemplates(t *testing.T) {
	templates, err := getTemplates()
	require.NoError(t, err)
	require.NotNil(t, templates)
	logoutTemplate := templates.Lookup("logout")
	require.NotNil(t, logoutTemplate)
}

func TestLogoutTemplate(t *testing.T) {
	templates, err := getTemplates()
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	data := map[string]any{
		"redirectURL": "http://example.org/",
		"providers": map[string]any{
			"renku": map[string]string{
				"baseURL":   "http://renku.org",
				"logoutURL": "http://renku.org/logout",
			},
		},
	}
	err = templates.ExecuteTemplate(buf, "logout", data)
	require.NoError(t, err)
	html := buf.String()
	// assert.Equal(t, "", html)
	assert.True(t, len(html) > 0)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<a class=\"btn-rk-green\" href=\"http://example.org/\">")
	assert.Contains(t, html, "<iframe id=\"logout-page-renku\" src=\"http://renku.org/logout\"></iframe>")
}
