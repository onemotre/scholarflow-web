package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
)

//go:embed templates/*.tmpl
var templatesEmbed embed.FS

// funcMap holds template helpers. dict builds a map from alternating key/value
// args so a sub-template can be invoked with several named parameters.
var funcMap = template.FuncMap{
	"dict": func(values ...any) (map[string]any, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict: odd number of arguments")
		}
		m := make(map[string]any, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict: key %v is not a string", values[i])
			}
			m[key] = values[i+1]
		}
		return m, nil
	},
}

// Each page is parsed in its OWN set (base + that page) so the shared "title"
// and "content" template names don't collide across pages.
func parsePage(page string) *template.Template {
	return template.Must(template.New(page).Funcs(funcMap).ParseFS(templatesEmbed, "templates/base.tmpl", "templates/"+page))
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
