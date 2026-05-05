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

type uploadedFile struct {
	Name  string
	Size  int64
	Error string
}

type uploadResult struct {
	Path  string
	Files []uploadedFile
	Error string // form-level error before any file is processed
	AllOK bool
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
		m.renderUploadError(w, "", fmt.Sprintf("parse form: %v", err))
		return
	}
	rel := strings.Trim(r.FormValue("path"), "/")
	dir, err := m.resolve(rel)
	if err != nil {
		m.renderUploadError(w, rel, err.Error())
		return
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		m.renderUploadError(w, rel, "target is not a directory")
		return
	}

	headers := r.MultipartForm.File["file"]
	if len(headers) == 0 {
		m.renderUploadError(w, rel, "no files selected")
		return
	}

	result := uploadResult{Path: rel, AllOK: true}
	for _, header := range headers {
		f, err := header.Open()
		if err != nil {
			result.Files = append(result.Files, uploadedFile{Name: header.Filename, Error: err.Error()})
			result.AllOK = false
			continue
		}
		written, err := m.saveFile(dir, header.Filename, f)
		f.Close()
		if err != nil {
			result.Files = append(result.Files, uploadedFile{Name: header.Filename, Error: err.Error()})
			result.AllOK = false
		} else {
			result.Files = append(result.Files, uploadedFile{Name: header.Filename, Size: written})
		}
	}

	// Refresh SELinux labels so Jellyfin's container can read newly added files.
	_ = exec.Command("chcon", "-Rt", "container_file_t", m.root).Run()

	_ = m.renderer.RenderPartial(w, "upload-result.html", result)
}

// saveFile writes a single uploaded file into baseDir, creating any intermediate
// subdirectories encoded in rawName (as sent by browsers for webkitdirectory uploads).
func (m *Module) saveFile(baseDir, rawName string, src io.Reader) (int64, error) {
	parts, err := sanitizeRelPath(rawName)
	if err != nil {
		return 0, err
	}

	destDir := baseDir
	if len(parts) > 1 {
		destDir = filepath.Join(append([]string{baseDir}, parts[:len(parts)-1]...)...)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return 0, fmt.Errorf("mkdir: %w", err)
		}
	}

	dest := filepath.Join(destDir, parts[len(parts)-1])
	tmp, err := os.CreateTemp(destDir, ".upload-*")
	if err != nil {
		return 0, fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	written, copyErr := io.Copy(tmp, src)
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		if copyErr != nil {
			return 0, copyErr
		}
		return 0, closeErr
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("rename: %w", err)
	}
	return written, nil
}

// sanitizeRelPath splits a browser-supplied filename (which may contain forward
// or back slashes for webkitdirectory uploads) into clean path components,
// rejecting any component that could escape the destination directory.
func sanitizeRelPath(rawName string) ([]string, error) {
	rawName = strings.ReplaceAll(rawName, "\\", "/")
	parts := strings.Split(rawName, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "." {
			continue
		}
		if p == ".." || strings.ContainsAny(p, "/\\") {
			return nil, errors.New("invalid path component")
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, errors.New("invalid filename")
	}
	return out, nil
}

func (m *Module) renderUploadError(w http.ResponseWriter, path, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	_ = m.renderer.RenderPartial(w, "upload-result.html", uploadResult{
		Path: path, Error: msg,
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
