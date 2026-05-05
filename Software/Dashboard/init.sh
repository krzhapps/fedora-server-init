#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$SCRIPT_DIR/app"
STATIC_DIR="$APP_DIR/static"

TAILWIND_VERSION="v3.4.17"
HTMX_VERSION="2.0.4"
BIN_PATH="/usr/local/bin/server-dashboard"
TAILWIND_BIN="/usr/local/bin/tailwindcss"

sudo dnf install -y curl

# Tailwind standalone CLI (no node dep)
if [ ! -x "$TAILWIND_BIN" ]; then
  echo "==> Installing tailwindcss $TAILWIND_VERSION"
  sudo curl -sSfL \
    "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-x64" \
    -o "$TAILWIND_BIN"
  sudo chmod +x "$TAILWIND_BIN"
fi

# Vendor htmx (gitignored — fetched at install time)
if [ ! -f "$STATIC_DIR/htmx.min.js" ]; then
  echo "==> Fetching htmx $HTMX_VERSION"
  curl -sSfL "https://unpkg.com/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" \
    -o "$STATIC_DIR/htmx.min.js"
fi

echo "==> Building Tailwind stylesheet"
(cd "$APP_DIR" && "$TAILWIND_BIN" \
  -c ./static/tailwind.config.js \
  -i ./static/input.css \
  -o ./static/styles.css \
  --minify)

echo "==> Building dashboard binary"
(cd "$APP_DIR" && go build -o /tmp/server-dashboard .)
sudo install -m 0755 /tmp/server-dashboard "$BIN_PATH"
rm -f /tmp/server-dashboard

echo "==> Installing systemd unit"
sudo install -m 0644 "$SCRIPT_DIR/dashboard.service" /etc/systemd/system/dashboard.service
sudo systemctl daemon-reload
sudo systemctl enable --now dashboard.service
sudo systemctl restart dashboard.service

echo "==> Opening firewall port 8090/tcp"
sudo firewall-cmd --permanent --add-port=8090/tcp
sudo firewall-cmd --reload

echo "==> Dashboard ready on http://$(hostname -I | awk '{print $1}'):8090"
