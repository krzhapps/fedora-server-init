# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Purpose

This is a Fedora Linux server setup automation repository. It provisions a fresh machine by installing and configuring software automatically. Tested on Fedora Server 43.

## Running Setup

```bash
# Full setup (runs all scripts in order)
bash ~/Setup/init.sh

# Individual scripts
bash ~/Setup/Software/Git/init.sh
bash ~/Setup/Software/Tailscale/init.sh
bash ~/Setup/Software/Docker/init.sh
bash ~/Setup/Software/Firewall/init.sh
bash ~/Setup/Software/Uv/init.sh
bash ~/Setup/Software/Tmux/init.sh
bash ~/Setup/Software/Golang/init.sh
bash ~/Setup/Software/Jellyfin/init.sh

# Update a specific service
bash ~/Setup/Software/Jellyfin/update.sh
```

## Architecture

- `init.sh` — Entry point; resolves its own directory via `BASH_SOURCE` and calls each script in `Software/` sequentially with strict error handling (`set -euo pipefail`).
- `Software/` — One subdirectory per tool. Each subdirectory contains:
  - `init.sh` — installs and configures the tool (required)
  - `update.sh` — updates the tool to the latest version (optional, only where relevant)

## Extending

Add a new setup step by creating `Software/YourTool/init.sh` and registering it in `init.sh`:

```bash
bash "$SCRIPT_DIR/Software/YourTool/init.sh"
```

All scripts must:
- Start with `#!/usr/bin/env bash`
- Use `set -euo pipefail`
- Be self-contained (handle their own firewall rules, dependencies, etc.)

## Platform

Fedora/RHEL only — scripts use `dnf` and `rpm`. Some scripts require `sudo`.
