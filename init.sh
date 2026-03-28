sudo dnf install -y git

# Create an SSH key that you can use for github, you need to add it to your account
ssh-keygen -t ed25519 -C "dkrzhalovski.apps@gmail.com"

# After installing you still need to login to your network (sudo tailscale up)
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up --advertise-tags=tag:server --ssh
