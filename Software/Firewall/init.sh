#!/usr/bin/env bash
set -euo pipefail

# Allow common development ports
sudo firewall-cmd --permanent --add-port=8000/tcp --add-port=3000/tcp
sudo firewall-cmd --reload
