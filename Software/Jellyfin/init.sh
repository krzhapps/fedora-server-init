#!/usr/bin/env bash
set -euo pipefail

sudo dnf5 install -y podman

sudo firewall-cmd --permanent --add-port=8096/tcp
sudo firewall-cmd --reload

sudo mkdir -p ~/jellyfin/{cache,config,media}

sudo podman run -d \
  --name jellyfin \
  --net=host \
  --restart=always \
  -v ~/jellyfin/config:/config:Z \
  -v ~/jellyfin/cache:/cache:Z \
  -v ~/jellyfin/media:/media:ro,z \
  docker.io/jellyfin/jellyfin:latest

sudo systemctl enable --now podman-restart.service
