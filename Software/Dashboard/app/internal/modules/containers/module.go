package containers

import (
	"net/http"

	"github.com/krzhalovski/fedora-server-init/dashboard/internal/server"
)

type Module struct {
	renderer *server.Renderer
}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string  { return "Containers" }
func (m *Module) Panel() string { return "containers.html" }

func (m *Module) UseRenderer(r *server.Renderer) {
	m.renderer = r
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /containers/rows", m.handleRows)
}

type viewData struct {
	Containers []Container
	Error      string
}

func (m *Module) handleRows(w http.ResponseWriter, r *http.Request) {
	data := viewData{}
	cs, err := List(r.Context())
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Containers = cs
	}
	if err := m.renderer.RenderPartial(w, "containers.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
