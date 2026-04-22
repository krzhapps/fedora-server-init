sudo dnf5 install podman -y

sudo firewall-cmd --add-port=8096/tcp --permanent
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
