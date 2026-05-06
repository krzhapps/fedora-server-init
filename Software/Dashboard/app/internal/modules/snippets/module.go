package snippets

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/krzhalovski/fedora-server-init/dashboard/internal/server"
)

type Snippet struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Module struct {
	filePath string
	renderer *server.Renderer
	mu       sync.Mutex
}

func New(filePath string) *Module {
	return &Module{filePath: filePath}
}

func (m *Module) Name() string  { return "Snippets" }
func (m *Module) Panel() string { return "snippets.html" }

func (m *Module) UseRenderer(r *server.Renderer) {
	m.renderer = r
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /snippets", m.handleList)
	mux.HandleFunc("POST /snippets", m.handleCreate)
}

type viewData struct {
	Snippets []Snippet
	Error    string
}

func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	snippets, err := m.readSnippets()
	data := viewData{}
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Snippets = recent(snippets, 5)
	}
	if renderErr := m.renderer.RenderPartial(w, "snippets.html", data); renderErr != nil {
		http.Error(w, renderErr.Error(), http.StatusInternalServerError)
	}
}

func (m *Module) handleCreate(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content")
	var snippets []Snippet
	var err error
	if content != "" {
		snippets, err = m.addSnippet(content)
	} else {
		snippets, err = m.readSnippets()
	}
	data := viewData{}
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Snippets = recent(snippets, 5)
	}
	if renderErr := m.renderer.RenderPartial(w, "snippets.html", data); renderErr != nil {
		http.Error(w, renderErr.Error(), http.StatusInternalServerError)
	}
}

func (m *Module) readSnippets() ([]Snippet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.load()
}

func (m *Module) addSnippet(content string) ([]Snippet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, err := m.load()
	if err != nil {
		return nil, err
	}
	updated := append([]Snippet{{
		ID:        strconv.FormatInt(time.Now().UnixNano(), 10),
		Content:   content,
		CreatedAt: time.Now(),
	}}, existing...)
	return updated, m.save(updated)
}

func (m *Module) load() ([]Snippet, error) {
	data, err := os.ReadFile(m.filePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snippets []Snippet
	if err := json.Unmarshal(data, &snippets); err != nil {
		return nil, err
	}
	return snippets, nil
}

func (m *Module) save(snippets []Snippet) error {
	data, err := json.Marshal(snippets)
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0600)
}

func recent(snippets []Snippet, n int) []Snippet {
	if len(snippets) <= n {
		return snippets
	}
	return snippets[:n]
}
