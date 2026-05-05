#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Starting system setup..."

bash "$SCRIPT_DIR/Software/Git/init.sh"
bash "$SCRIPT_DIR/Software/Tailscale/init.sh"
bash "$SCRIPT_DIR/Software/Docker/init.sh"
bash "$SCRIPT_DIR/Software/Firewall/init.sh"
bash "$SCRIPT_DIR/Software/Uv/init.sh"
bash "$SCRIPT_DIR/Software/Tmux/init.sh"
bash "$SCRIPT_DIR/Software/Golang/init.sh"
bash "$SCRIPT_DIR/Software/Jellyfin/init.sh"

echo "==> Setup complete."
