#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
DOCKER_BIN="${DOCKER_BIN:-docker}"
RETENTION_UNTIL="${IMAGE_CLEANUP_UNTIL:-168h}"
UPDATE_LOCK_DIR="${AUTO_UPDATE_LOCK_DIR:-$ROOT_DIR/.git/mypaas-update.lock}"
MODE="dry-run"

usage() {
  cat <<'EOF'
Usage: scripts/docker-retention.sh [--dry-run|--apply]

Default is --dry-run. The command inventories MyPaas-managed images, Docker
storage, BuildKit cache, and MyPaas artifact directories. --apply runs the same
age-scoped managed-image and BuildKit prune used by the scheduled cleanup.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) MODE="dry-run" ;;
    --apply) MODE="apply" ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

if [[ ! "$RETENTION_UNTIL" =~ ^[0-9]+(s|m|h|d|w)$ ]]; then
  echo "IMAGE_CLEANUP_UNTIL must be a Docker duration such as 168h or 7d." >&2
  exit 2
fi

run_docker() {
  "$DOCKER_BIN" "$@"
}

section() {
  printf '\n=== %s ===\n' "$1"
}

section "retention policy"
printf 'mode=%s\n' "$MODE"
printf 'managed_image_label=mypaas.managed=true\n'
printf 'until=%s\n' "$RETENTION_UNTIL"
printf 'update_lock=%s\n' "$UPDATE_LOCK_DIR"

section "docker storage"
run_docker system df || true

section "managed image inventory"
run_docker image ls --filter 'label=mypaas.managed=true' --digests || true

section "running container images"
run_docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' || true

section "buildkit cache"
run_docker builder du || true

section "mypaas artifacts"
for path in /var/lib/mypaas/static /var/lib/mypaas/backups /var/lib/mypaas/compose /var/lib/mypaas/volumes; do
  if [[ -e "$path" ]]; then
    du -sh "$path" || true
  else
    printf '%s\tmissing\n' "$path"
  fi
done

if [[ "$MODE" == "dry-run" ]]; then
  section "planned cleanup"
  printf 'docker image prune -a -f --filter label=mypaas.managed=true --filter until=%s\n' "$RETENTION_UNTIL"
  printf 'docker builder prune -f --filter until=%s\n' "$RETENTION_UNTIL"
  printf 'No data was deleted. Re-run with --apply on a controlled host after reviewing this inventory.\n'
  exit 0
fi

if [[ -d "$UPDATE_LOCK_DIR" ]]; then
  echo "Refusing cleanup while the MyPaas updater lock exists: $UPDATE_LOCK_DIR" >&2
  exit 1
fi

section "managed image cleanup"
run_docker image prune -a -f --filter 'label=mypaas.managed=true' --filter "until=$RETENTION_UNTIL"

section "buildkit cleanup"
run_docker builder prune -f --filter "until=$RETENTION_UNTIL"

section "post-cleanup docker storage"
run_docker system df || true
