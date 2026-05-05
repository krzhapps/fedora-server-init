package server

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
)

var funcMap = template.FuncMap{
	"humanSize": humanSize,
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// Renderer holds parsed templates. Pages share layout.html; partials are
// rendered standalone for HTMX swaps.
type Renderer struct {
	pages    *template.Template
	partials *template.Template
	static   fs.FS
}

func NewRenderer(templatesFS, staticDir fs.FS) (*Renderer, error) {
	pages, err := template.New("").Funcs(funcMap).ParseFS(templatesFS,
		"layout.html",
		"index.html",
		"partials/*.html",
	)
	if err != nil {
		return nil, fmt.Errorf("parse pages: %w", err)
	}
	partials, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}
	return &Renderer{pages: pages, partials: partials, static: staticDir}, nil
}

func (r *Renderer) RenderPage(w io.Writer, name string, data any) error {
	if r.pages.Lookup(name) == nil {
		return fmt.Errorf("page template not found: %s", name)
	}
	return r.pages.ExecuteTemplate(w, "layout.html", struct {
		Body string
		Data any
	}{Body: name, Data: data})
}

// RenderPartial renders a single partial template (for HTMX fragments).
func (r *Renderer) RenderPartial(w io.Writer, name string, data any) error {
	if r.partials.Lookup(name) == nil {
		return fmt.Errorf("partial template not found: %s", name)
	}
	return r.partials.ExecuteTemplate(w, name, data)
}

func (r *Renderer) Static() fs.FS {
	return r.static
}
