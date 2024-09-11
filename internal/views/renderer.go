package views

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	templates *template.Template
}

func (tr *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return tr.templates.ExecuteTemplate(w, name, data)
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
