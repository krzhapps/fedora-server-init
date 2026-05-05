#!/usr/bin/env bash
set -euo pipefail

sudo dnf install -y git

# Create an SSH key to use with GitHub — add the public key to your account afterwards
ssh-keygen -t ed25519 -C "dkrzhalovski.apps@gmail.com"
