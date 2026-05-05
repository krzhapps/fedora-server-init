#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$SCRIPT_DIR/app"
BIN_PATH="/usr/local/bin/server-dashboard"
TAILWIND_BIN="/usr/local/bin/tailwindcss"

echo "==> Rebuilding Tailwind stylesheet"
( cd "$APP_DIR" && "$TAILWIND_BIN" \
    -c ./static/tailwind.config.js \
    -i ./static/input.css \
    -o ./static/styles.css \
    --minify )

echo "==> Rebuilding dashboard binary"
( cd "$APP_DIR" && go build -o /tmp/server-dashboard . )
sudo install -m 0755 /tmp/server-dashboard "$BIN_PATH"
rm -f /tmp/server-dashboard

echo "==> Restarting service"
sudo systemctl restart dashboard.service
sudo systemctl status --no-pager dashboard.service
