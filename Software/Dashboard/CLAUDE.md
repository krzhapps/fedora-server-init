# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Go web dashboard for a Fedora home server. It surfaces Podman container status and a Jellyfin media file browser. Served as a systemd service on port 8090.

## Build

Both steps are required before deploying — templates and static assets are embedded at compile time via `//go:embed`.

```bash
# 1. Compile Tailwind CSS (standalone CLI, no Node required)
cd app
tailwindcss -c ./static/tailwind.config.js -i ./static/input.css -o ./static/styles.css --minify

# 2. Build the binary
go build -o /tmp/server-dashboard .
```

`update.sh` runs both steps and restarts the service. `init.sh` also downloads the Tailwind CLI and htmx if missing.

## Dev Workflow

```bash
# Run locally (media root defaults to /root/jellyfin/media)
MEDIA_ROOT=/your/path LISTEN_ADDR=:8090 go run ./app

# Tailwind watch mode during template editing
cd app && tailwindcss -c ./static/tailwind.config.js -i ./static/input.css -o ./static/styles.css --watch
```

Note: `static/htmx.min.js` and `static/styles.css` are gitignored and generated at install time.

## Architecture

### Module System

The central extension point is `server.Module` (`internal/server/server.go`):

```go
type Module interface {
    Name() string          // display name in the UI
    Routes(*http.ServeMux) // register HTTP handlers
    Panel() string         // partial template name, e.g. "containers.html"
}
```

Register modules in `main.go`. Modules that need to render HTML implement `rendererInjector` (an unexported interface) to receive the shared `*server.Renderer` via `UseRenderer`.

### Rendering

`Renderer` (`internal/server/render.go`) has two modes:
- `RenderPage` — full page wrapped in `layout.html` (used only by `GET /`)
- `RenderPartial` — standalone partial for HTMX fragment swaps (used by all module handlers)

All partials live in `templates/partials/`. Each module handler renders its own partial directly in response to HTMX requests.

### Existing Modules

- **containers** (`internal/modules/containers/`) — calls `podman ps -a --format=json` and renders a table.
- **files** (`internal/modules/files/`) — browsable tree rooted at `MEDIA_ROOT`. Handles single-file and directory (`webkitdirectory`) uploads via temp-file-then-rename, and `mkdir`. Path traversal is blocked in `resolve()`. After writes, re-applies SELinux labels with `chcon -Rt container_file_t`.

### Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `LISTEN_ADDR` | `:8090` | TCP listen address |
| `MEDIA_ROOT` | `/root/jellyfin/media` | Root for the files module |

The systemd unit (`dashboard.service`) overrides `MEDIA_ROOT` to `/home/dkrzhalovski/jellyfin/media`.

## Adding a New Module

1. Create `internal/modules/yourmodule/module.go` implementing `server.Module`.
2. Implement `UseRenderer(*server.Renderer)` if the module renders HTML.
3. Add a partial template at `templates/partials/yourmodule.html`.
4. Register in `main.go`: `modules.New()` inside the `Modules` slice.
