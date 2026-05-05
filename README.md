# Setup
Fedora server setup automation. Provisions a fresh machine by installing and configuring software automatically. Tested only on Fedora Server 43. For the workstation version with more features and a better desktop environment look at: https://github.com/krzhapps/fedora-init

## Software
1. Git
2. Tailscale
3. Docker
4. Firewall
5. Uv
6. Tmux
7. Golang
8. Jellyfin

## Usage

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
