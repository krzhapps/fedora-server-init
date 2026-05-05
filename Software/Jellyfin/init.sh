#!/usr/bin/env bash
set -euo pipefail

sudo dnf5 install -y podman

sudo firewall-cmd --permanent --add-port=8096/tcp --add-port=8080/tcp
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

mkdir -p ~/filebrowser
touch ~/filebrowser/filebrowser.db

sudo podman run -d \
  --name media-filebrowser \
  -p 8080:8080 \
  --restart=always \
  -v ~/jellyfin/media:/srv:z \
  -v ~/filebrowser:/database:Z \
  docker.io/filebrowser/filebrowser:latest \
  --port 8080

# Run: sudo podman logs media-filebrowser — to retrieve the auto-generated password

sudo systemctl enable --now podman-restart.service
