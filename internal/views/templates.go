package views

import (
	"embed"
	"html/template"
)

//go:embed templates/*
var embedFS embed.FS

func getTemplates() (*template.Template, error) {
	return template.ParseFS(embedFS, "**/*.html")
}
