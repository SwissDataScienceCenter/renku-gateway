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
		"renkuBaseURL": "http://renku.ch",
		"redirectURL":  "http://example.org/",
		"providers": map[string]any{
			"renku": map[string]string{
				"logoutURL": "http://renku.ch/logout",
			},
		},
	}
	err = templates.ExecuteTemplate(buf, "logout", data)
	require.NoError(t, err)
	html := buf.String()
	assert.True(t, len(html) > 0)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<link rel=\"stylesheet\" href=\"http://renku.ch/static/public/theme.css\">")
	assert.Contains(t, html, "<a class=\"btn-rk-green\" href=\"http://example.org/\">")
	assert.Contains(t, html, "<iframe id=\"logout-page-renku\" src=\"http://renku.ch/logout\" style=\"display:none;\"></iframe>")
}

func TestGitlabLogoutTemplate(t *testing.T) {
	templates, err := getTemplates()
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	data := map[string]any{
		"logoutURL": "http://example.org/logout",
	}
	err = templates.ExecuteTemplate(buf, "gitlab_logout", data)
	require.NoError(t, err)
	html := buf.String()
	assert.True(t, len(html) > 0)
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "action=\"http://example.org/logout\"")
}
