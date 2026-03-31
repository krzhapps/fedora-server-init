sudo dnf install -y git

# Create an SSH key that you can use for github, you need to add it to your account
ssh-keygen -t ed25519 -C "dkrzhalovski.apps@gmail.com"

# After installing you still need to login to your network (sudo tailscale up)
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up --advertise-tags=tag:server --ssh

# Install and enable docker
sudo dnf config-manager addrepo --from-repofile=https://download.docker.com/linux/fedora/docker-ce.repo
sudo dnf install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker dkrzhalovski

# Allow common front-end ports
sudo firewall-cmd --permanent --add-port=8000/tcp --add-port=3000/tcp
