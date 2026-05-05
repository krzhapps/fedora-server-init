package server

import (
	"io/fs"
	"net/http"
)

// Module is the extension point for dashboard panels. Each module owns its
// HTTP routes and a name shown in the UI. Add a new feature by implementing
// this interface and registering it in main.go.
type Module interface {
	Name() string
	Routes(mux *http.ServeMux)
	// Panel returns the template name (under templates/partials/) used to
	// render the module's section on the index page.
	Panel() string
}

type Config struct {
	Templates fs.FS
	Static    fs.FS
	Modules   []Module
}

type Server struct {
	modules  []Module
	renderer *Renderer
	mux      *http.ServeMux
}

func New(cfg Config) (*Server, error) {
	r, err := NewRenderer(cfg.Templates, cfg.Static)
	if err != nil {
		return nil, err
	}
	s := &Server{modules: cfg.Modules, renderer: r, mux: http.NewServeMux()}

	for _, m := range cfg.Modules {
		if injector, ok := m.(rendererInjector); ok {
			injector.UseRenderer(r)
		}
		m.Routes(s.mux)
	}

	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(cfg.Static)))
	s.mux.HandleFunc("GET /{$}", s.handleIndex)
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

type indexData struct {
	Modules []Module
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.renderer.RenderPage(w, "index.html", indexData{Modules: s.modules}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// rendererInjector lets modules receive the shared renderer without making
// it part of the public Module contract.
type rendererInjector interface {
	UseRenderer(*Renderer)
}
