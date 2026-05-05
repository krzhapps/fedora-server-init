package files

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/krzhalovski/fedora-server-init/dashboard/internal/server"
)

type Module struct {
	root     string
	renderer *server.Renderer
}

func New(root string) *Module {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return &Module{root: filepath.Clean(abs)}
}

func (m *Module) Name() string  { return "Media" }
func (m *Module) Panel() string { return "files.html" }

func (m *Module) UseRenderer(r *server.Renderer) {
	m.renderer = r
}

func (m *Module) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /files", m.handleList)
	mux.HandleFunc("POST /files/upload", m.handleUpload)
	mux.HandleFunc("POST /files/mkdir", m.handleMkdir)
}

type Entry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

type viewData struct {
	Path    string  // user-facing relative path within root, "" for root
	Parent  string  // parent dir path, "" if at root
	AtRoot  bool
	Entries []Entry
	Error   string
}

type uploadResult struct {
	Path    string
	Name    string
	Size    int64
	Error   string
}

// resolve returns the absolute filesystem path for a user-supplied relative
// path, ensuring it stays inside the configured root. Returns an error for
// any escape attempt.
func (m *Module) resolve(rel string) (string, error) {
	rel = strings.TrimPrefix(rel, "/")
	joined := filepath.Join(m.root, rel)
	clean := filepath.Clean(joined)
	if clean != m.root && !strings.HasPrefix(clean, m.root+string(os.PathSeparator)) {
		return "", errors.New("path escapes media root")
	}
	if real, err := filepath.EvalSymlinks(clean); err == nil {
		if real != m.root && !strings.HasPrefix(real, m.root+string(os.PathSeparator)) {
			return "", errors.New("path escapes media root via symlink")
		}
	}
	return clean, nil
}

func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	rel := strings.Trim(r.URL.Query().Get("path"), "/")
	abs, err := m.resolve(rel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := viewData{Path: rel, AtRoot: rel == ""}
	if !data.AtRoot {
		data.Parent = filepath.Dir(rel)
		if data.Parent == "." {
			data.Parent = ""
		}
	}

	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		data.Error = err.Error()
		_ = m.renderer.RenderPartial(w, "files.html", data)
		return
	}

	for _, e := range dirEntries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		data.Entries = append(data.Entries, Entry{
			Name:    e.Name(),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(data.Entries, func(i, j int) bool {
		a, b := data.Entries[i], data.Entries[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	if err := m.renderer.RenderPartial(w, "files.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const maxUploadBytes = 50 << 30 // 50 GiB; media files are large

func (m *Module) handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		m.renderUploadError(w, "", "", fmt.Sprintf("parse form: %v", err))
		return
	}
	rel := strings.Trim(r.FormValue("path"), "/")
	dir, err := m.resolve(rel)
	if err != nil {
		m.renderUploadError(w, rel, "", err.Error())
		return
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		m.renderUploadError(w, rel, "", "target is not a directory")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		m.renderUploadError(w, rel, "", fmt.Sprintf("read file: %v", err))
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename)
	if name == "" || name == "." || name == "/" || strings.Contains(name, string(os.PathSeparator)) {
		m.renderUploadError(w, rel, header.Filename, "invalid filename")
		return
	}

	dest := filepath.Join(dir, name)
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		m.renderUploadError(w, rel, name, fmt.Sprintf("temp file: %v", err))
		return
	}
	tmpPath := tmp.Name()
	written, copyErr := io.Copy(tmp, file)
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		msg := "copy failed"
		if copyErr != nil {
			msg = copyErr.Error()
		} else if closeErr != nil {
			msg = closeErr.Error()
		}
		m.renderUploadError(w, rel, name, msg)
		return
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		_ = os.Remove(tmpPath)
		m.renderUploadError(w, rel, name, fmt.Sprintf("rename: %v", err))
		return
	}

	// Refresh SELinux labels so Jellyfin's container can read newly added files.
	_ = exec.Command("chcon", "-Rt", "container_file_t", m.root).Run()

	_ = m.renderer.RenderPartial(w, "upload-result.html", uploadResult{
		Path: rel, Name: name, Size: written,
	})
}

func (m *Module) renderUploadError(w http.ResponseWriter, path, name, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = m.renderer.RenderPartial(w, "upload-result.html", uploadResult{
		Path: path, Name: name, Error: msg,
	})
}

type mkdirResult struct {
	Path  string
	Name  string
	Error string
}

func (m *Module) handleMkdir(w http.ResponseWriter, r *http.Request) {
	rel := strings.Trim(r.FormValue("path"), "/")
	name := filepath.Base(strings.TrimSpace(r.FormValue("name")))
	if name == "" || name == "." || strings.Contains(name, string(os.PathSeparator)) {
		m.renderMkdirError(w, rel, name, "invalid directory name")
		return
	}
	dir, err := m.resolve(rel)
	if err != nil {
		m.renderMkdirError(w, rel, name, err.Error())
		return
	}
	dest := filepath.Join(dir, name)
	if err := os.Mkdir(dest, 0o755); err != nil {
		m.renderMkdirError(w, rel, name, err.Error())
		return
	}
	_ = exec.Command("chcon", "-Rt", "container_file_t", m.root).Run()
	_ = m.renderer.RenderPartial(w, "mkdir-result.html", mkdirResult{Path: rel, Name: name})
}

func (m *Module) renderMkdirError(w http.ResponseWriter, path, name, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = m.renderer.RenderPartial(w, "mkdir-result.html", mkdirResult{
		Path: path, Name: name, Error: msg,
	})
}
