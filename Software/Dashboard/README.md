# Server Dashboard

A lightweight web dashboard for a Fedora home server. Displays Podman container status and provides a Jellyfin media file browser.

## Features

- **Containers** — live table of all Podman containers (`podman ps -a`)
- **Media browser** — navigate, upload (single files or whole directories), and create folders under the configured media root

## Installation

```bash
bash init.sh
```

This installs the Tailwind CLI and htmx, builds the CSS and Go binary, installs a systemd unit, and opens port 8090 in the firewall.

## Updating

```bash
bash update.sh
```

Rebuilds the CSS and binary, then restarts the service.

## Configuration

The service reads two environment variables (set in `dashboard.service`):

| Variable | Default | Description |
|---|---|---|
| `LISTEN_ADDR` | `:8090` | TCP listen address |
| `MEDIA_ROOT` | `/root/jellyfin/media` | Root directory for the media browser |

## Tech Stack

- **Go** — standard library only, no external dependencies
- **HTMX** — partial page updates without a JS framework
- **Tailwind CSS** — standalone CLI (no Node.js required)
