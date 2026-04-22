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

mkdir ~/filebrowser
touch ~/filebrowser/filebrowser.db

sudo podman run -d \
  --name media-filebrowser \
  -p 8080:8080 \
  --restart=always \
  -v ~/jellyfin/media:/srv:z \
  -v ~/filebrowser:/database:Z \
  docker.io/filebrowser/filebrowser:latest \
  --port 8080

# sudo podman logs media-filebrowser # Use it to check the randomly generated password

sudo systemctl enable --now podman-restart.service
