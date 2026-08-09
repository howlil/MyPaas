#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

sudo_cmd() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

echo "WARNING: This will completely destroy the MyPaas installation on this VM."
echo "All projects, databases, configurations, and backups will be PERMANENTLY DELETED."
read -p "Are you sure you want to continue? Type 'DESTROY' to confirm: " confirm

if [[ "$confirm" != "DESTROY" ]]; then
  echo "Aborted."
  exit 0
fi

cd "$ROOT_DIR"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  source "$ENV_FILE"
  set +a
fi

PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-prod}"

docker_prefix() {
  if docker ps >/dev/null 2>&1; then
    printf 'docker'
    return
  fi
  if command -v sudo >/dev/null 2>&1 && sudo docker ps >/dev/null 2>&1; then
    printf 'sudo docker'
    return
  fi
  die "current user cannot access Docker."
}

DOCKER_BIN="$(docker_prefix)"
COMPOSE_BIN="$DOCKER_BIN compose"

log "Stopping and removing MyPaas core services and images..."
if [[ -f "$COMPOSE_FILE" ]]; then
  if [[ -f "$ENV_FILE" ]]; then
    $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" down -v --rmi all --remove-orphans || true
  else
    $COMPOSE_BIN -f "$COMPOSE_FILE" down -v --rmi all --remove-orphans || true
  fi
fi

log "Stopping and removing all user project containers in network $PROJECT_NETWORK..."
if $DOCKER_BIN network inspect "$PROJECT_NETWORK" >/dev/null 2>&1; then
  containers=$($DOCKER_BIN network inspect "$PROJECT_NETWORK" --format '{{range .Containers}}{{.Name}} {{end}}' | xargs)
  if [[ -n "$containers" ]]; then
    log "Removing containers: $containers"
    $DOCKER_BIN rm -f $containers || true
  fi
  log "Removing docker network $PROJECT_NETWORK..."
  $DOCKER_BIN network rm "$PROJECT_NETWORK" || true
fi

log "Removing host directories..."
for dir in \
  /var/lib/mypaas \
  /tmp/mypaas/builds
do
  if [[ -d "$dir" ]]; then
    sudo_cmd rm -rf "$dir"
  fi
done

log "Removing .env file..."
if [[ -f "$ENV_FILE" ]]; then
  rm -f "$ENV_FILE"
fi

log "Uninstall complete. MyPaas has been totally destroyed."

# Change directory before deleting the source folder to prevent errors
cd /
if [[ -d "$ROOT_DIR" && "$ROOT_DIR" != "/" ]]; then
  echo "==> Removing MyPaas source folder ($ROOT_DIR)..."
  sudo_cmd rm -rf "$ROOT_DIR"
fi

