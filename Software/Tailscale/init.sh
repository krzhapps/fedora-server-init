#!/usr/bin/env bash
set -euo pipefail

curl -fsSL https://tailscale.com/install.sh | sh

# After installing, log in to your network with: sudo tailscale up
sudo tailscale up --advertise-tags=tag:server --ssh
