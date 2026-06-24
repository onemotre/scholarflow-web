package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed templates/*.tmpl
var templatesEmbed embed.FS

// Each page is parsed in its OWN set (base + that page) so the shared "title"
// and "content" template names don't collide across pages.
func parsePage(page string) *template.Template {
	return template.Must(template.ParseFS(templatesEmbed, "templates/base.tmpl", "templates/"+page))
}

var pages = map[string]*template.Template{
	"collection.tmpl": parsePage("collection.tmpl"),
	"paper.tmpl":      parsePage("paper.tmpl"),
	"error.tmpl":      parsePage("error.tmpl"),
}

// Render executes the "base" template of the named page's set.
func Render(w io.Writer, name string, data any) error {
	tmpl, ok := pages[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}
