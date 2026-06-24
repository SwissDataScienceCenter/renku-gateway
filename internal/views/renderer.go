package views

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v5"
)

type TemplateRenderer struct {
	templates *template.Template
}

func (tr *TemplateRenderer) Render(c *echo.Context, w io.Writer, templateName string, data any) error {
	return tr.templates.ExecuteTemplate(w, templateName, data)
}

func (tr *TemplateRenderer) Register(e *echo.Echo) {
	e.Renderer = tr
}

func NewTemplateRenderer() (*TemplateRenderer, error) {
	templates, err := getTemplates()
	if err != nil {
		return &TemplateRenderer{}, err
	}
	tr := TemplateRenderer{
		templates,
	}
	return &tr, nil
}
