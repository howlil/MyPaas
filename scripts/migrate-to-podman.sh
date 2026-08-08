#!/usr/bin/env bash
set -euo pipefail

SUDO=""
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
fi

echo "==========================================================="
echo " Migrating MyPaas from Docker Engine to Podman Engine... "
echo "==========================================================="

echo "[1/5] Stopping existing Docker daemon..."
$SUDO systemctl stop docker docker.socket || true

echo "[2/5] Uninstalling Docker Engine (keeping CLI & Compose Plugin)..."
$SUDO apt-get remove -y docker-ce containerd.io docker.io || true
$SUDO apt-get autoremove -y || true

echo "[3/5] Installing Podman & Docker CLI..."
$SUDO apt-get update
$SUDO apt-get install -y podman docker-ce-cli docker-compose-plugin

echo "[4/5] Enabling Podman Socket (Docker API compatibility)..."
$SUDO systemctl enable --now podman.socket

echo "[5/5] Bridging Docker Socket to Podman..."
$SUDO ln -sf /run/podman/podman.sock /var/run/docker.sock

echo "==========================================================="
echo " Verification:"
$SUDO docker info | grep -i -E "name|podman|engine" || true
echo "==========================================================="
echo "Migration complete!"
echo "NOTE: All your previous containers in Docker are gone (daemon changed)."
echo "Please run: bash scripts/deploy-to-vm.sh to boot MyPaas on Podman."
echo "Then, open the dashboard and click 'Deploy' on each of your projects."
