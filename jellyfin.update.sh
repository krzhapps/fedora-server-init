sudo podman pull docker.io/jellyfin/jellyfin:latest

sudo podman stop jellyfin
sudo podman rm jellyfin

sudo podman run -d \
  --name jellyfin \
  --net=host \
  --restart=always \
  -v ~/jellyfin/config:/config:Z \
  -v ~/jellyfin/cache:/cache:Z \
  -v ~/jellyfin/media:/media:ro,z \
  docker.io/jellyfin/jellyfin:latest
