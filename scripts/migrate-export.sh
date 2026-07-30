#!/usr/bin/env bash
# migrate-export.sh — Run on the OLD VM from the MyPaas project root.
# Creates a single archive containing everything needed to restore on a new VM.
set -euo pipefail

EXPORT_DIR="/tmp/mypaas-migration"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
ARCHIVE="/tmp/mypaas-export-${TIMESTAMP}.tar.gz"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*" >&2; exit 1; }

echo ""
echo "============================================"
echo "  MyPaas VM Migration — Export"
echo "============================================"
echo ""

# ── 1. Pre-flight ────────────────────────────────────────────────────
info "Pre-flight checks..."

command -v docker &>/dev/null   || fail "docker not found"
[ -f .env ]                     || fail ".env not found — run this from the MyPaas project root"

# shellcheck disable=SC1091
source .env

POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-mypaas-postgres-prod}"
DB_USER="${POSTGRES_USER:?POSTGRES_USER not set}"
DB_NAME="${POSTGRES_DB:?POSTGRES_DB not set}"

docker ps --format '{{.Names}}' | grep -q "^${POSTGRES_CONTAINER}$" \
    || fail "Postgres container '${POSTGRES_CONTAINER}' is not running"

ok "Pre-flight passed"

# ── 2. Prepare workspace ────────────────────────────────────────────
info "Preparing export workspace..."
rm -rf "$EXPORT_DIR"
mkdir -p "$EXPORT_DIR/databases"

# ── 3. Stop project containers (keep infra) ─────────────────────────
info "Stopping user project containers..."
PROJECT_CONTAINERS=$(docker ps --filter "label=mypaas.managed=true" -q 2>/dev/null || true)
if [ -n "$PROJECT_CONTAINERS" ]; then
    echo "$PROJECT_CONTAINERS" | xargs docker stop
    ok "Stopped project containers"
else
    ok "No running project containers"
fi

# ── 4. Dump MyPaas system database ──────────────────────────────────
info "Dumping MyPaas system database (${DB_NAME})..."
docker exec "$POSTGRES_CONTAINER" pg_dump \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --format=custom \
    --no-owner \
    --no-privileges \
    -f "/tmp/_mypaas_system.dump"
docker cp "${POSTGRES_CONTAINER}:/tmp/_mypaas_system.dump" "$EXPORT_DIR/databases/system.dump"
docker exec "$POSTGRES_CONTAINER" rm "/tmp/_mypaas_system.dump"
ok "System database dumped"

# ── 5. Dump shared project databases (mypaas_p_*) ───────────────────
info "Looking for shared project databases..."
PROJECT_DBS=$(docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -A -c \
    "SELECT datname FROM pg_database WHERE datname LIKE 'mypaas_p_%'" 2>/dev/null || true)

if [ -n "$PROJECT_DBS" ]; then
    DB_COUNT=$(echo "$PROJECT_DBS" | wc -l | tr -d ' ')
    info "Found ${DB_COUNT} shared project database(s), dumping..."
    while IFS= read -r dbname; do
        [ -z "$dbname" ] && continue
        info "  Dumping ${dbname}..."
        docker exec "$POSTGRES_CONTAINER" pg_dump \
            -U "$DB_USER" \
            -d "$dbname" \
            --format=custom \
            --no-owner \
            --no-privileges \
            -f "/tmp/_${dbname}.dump"
        docker cp "${POSTGRES_CONTAINER}:/tmp/_${dbname}.dump" "$EXPORT_DIR/databases/${dbname}.dump"
        docker exec "$POSTGRES_CONTAINER" rm "/tmp/_${dbname}.dump"
    done <<< "$PROJECT_DBS"

    # Also dump the roles used by shared databases
    docker exec "$POSTGRES_CONTAINER" pg_dumpall \
        -U "$DB_USER" \
        --roles-only \
        -f "/tmp/_roles.sql"
    docker cp "${POSTGRES_CONTAINER}:/tmp/_roles.sql" "$EXPORT_DIR/databases/roles.sql"
    docker exec "$POSTGRES_CONTAINER" rm "/tmp/_roles.sql"
    ok "Shared databases and roles dumped"
else
    ok "No shared project databases found"
fi

# ── 6. Copy .env ────────────────────────────────────────────────────
info "Copying .env..."
cp .env "$EXPORT_DIR/dot-env"
ok ".env copied"

# ── 7. Copy persistent directories ──────────────────────────────────
info "Copying persistent data..."

copy_dir() {
    local src="$1" name="$2"
    if [ -d "$src" ]; then
        size=$(du -sh "$src" 2>/dev/null | cut -f1)
        info "  ${name} (${size})..."
        cp -a "$src" "$EXPORT_DIR/$name"
        ok "  ${name} copied"
    else
        warn "  ${name} not found at ${src} — skipping"
    fi
}

copy_dir "/var/lib/mypaas/volumes" "volumes"
copy_dir "/var/lib/mypaas/compose" "compose"
copy_dir "/var/lib/mypaas/static"  "static"

# ── 8. Write manifest ───────────────────────────────────────────────
info "Writing manifest..."
cat > "$EXPORT_DIR/manifest.json" <<MANIFEST
{
  "version": 1,
  "exported_at": "$(date -Iseconds)",
  "hostname": "$(hostname)",
  "mypaas_db": "${DB_NAME}",
  "shared_dbs": [$(echo "$PROJECT_DBS" | sed '/^$/d' | awk '{printf "\"%s\",",$0}' | sed 's/,$//')]
}
MANIFEST
ok "Manifest written"

# ── 9. Create archive ───────────────────────────────────────────────
info "Creating archive (this may take a while)..."
tar czf "$ARCHIVE" -C "$EXPORT_DIR" .
rm -rf "$EXPORT_DIR"

SIZE=$(du -sh "$ARCHIVE" | cut -f1)

echo ""
echo "============================================"
echo -e "  ${GREEN}Export Complete!${NC}"
echo "============================================"
echo ""
echo "  Archive : $ARCHIVE"
echo "  Size    : $SIZE"
echo ""
echo "  Transfer to new VM:"
echo "    scp $ARCHIVE user@new-vm:/tmp/"
echo ""
echo "  Then on the new VM:"
echo "    cd /path/to/mypaas"
echo "    bash scripts/migrate-import.sh /tmp/$(basename "$ARCHIVE")"
echo ""
