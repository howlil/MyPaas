# Beta backup and restore drill

This runbook is the runtime evidence procedure for the `core/backup-restore` beta-readiness workstream. The repository implementation can be reviewed and unit-tested without a VM, but this gate remains `BLOCKED_ON_VM_EVIDENCE` until a bundle is restored onto a fresh controlled VM and the acceptance checks below pass.

## Scope

`scripts/backup-restore.py` creates a disaster-recovery bundle containing:

- a custom-format dump of the MyPaas control-plane PostgreSQL database;
- the production `.env` file under `private/config.env`;
- `/var/lib/mypaas/static`;
- `/var/lib/mypaas/compose`;
- Docker volumes explicitly labeled `mypaas.managed=true`;
- Docker Compose volumes whose Compose project belongs to an active MyPaas project.

The manifest records paths, sizes, checksums, counts, and the source Git SHA. It never records configuration values. **The bundle itself is sensitive** because `private/config.env` contains production secrets; keep the bundle on encrypted/restricted storage and do not attach it to public CI artifacts or pull requests.

## Source VM: create a consistent bundle

Inspect the plan first:

```bash
python3 scripts/backup-restore.py plan
```

A full bundle refuses to archive a managed volume while a running container is using it. For the controlled beta drill, explicitly quiesce those project containers for the short snapshot window:

```bash
sudo python3 scripts/backup-restore.py backup \
  --install-dir /opt/mypaas \
  --quiesce-managed-containers
```

The tool restarts any project containers it stopped, writes `manifest.json` and `backup-report.json`, and prints the bundle path. Preserve that path privately.

Verify it independently before moving to the target VM:

```bash
sudo python3 scripts/backup-restore.py verify --bundle /var/lib/mypaas/backups/full-...
```

A checksum failure is a failed drill. Do not continue with a damaged bundle.

## Fresh target VM preparation

Use a disposable VM with Docker and the normal MyPaas host prerequisites. Clone MyPaas and check out **the exact `sourceGitSha` from the bundle manifest**. The default restore path refuses to restore onto a different checkout.

Do not copy a newly generated `.env` over the bundle. The original encryption key and other production configuration are part of the recovery material and are required for encrypted environment values to remain readable.

Transfer the bundle through a private channel and preserve restrictive filesystem permissions.

## Restore

The restore command is destructive and therefore requires an explicit confirmation flag:

```bash
sudo python3 scripts/backup-restore.py restore \
  --bundle /secure/path/full-... \
  --install-dir /opt/mypaas \
  --confirm-restore
```

Before mutation the tool verifies every manifest checksum. It then restores production configuration, static/Compose data, managed volumes, starts the control-plane PostgreSQL service, and restores the PostgreSQL dump. An existing target `.env` is copied to a timestamped `.pre-restore-*` file before replacement.

After restore, start/reconcile the full platform from the same checkout and run normal production verification:

```bash
cd /opt/mypaas
sudo MYPAAS_IMAGE_TAG="$(git rev-parse HEAD)" bash scripts/deploy-to-vm.sh
sudo bash scripts/verify-production.sh
```

## Mandatory acceptance evidence

Record the target VM shape, bundle `sourceGitSha`, restore report, and timestamps. Then prove all of the following on the fresh VM:

- owner login succeeds;
- project inventory and deployment history match the source installation;
- encrypted project environment values are readable by the application without exposing their plaintext in the evidence;
- static project routes return expected content;
- container and Compose project routes recover;
- at least one restored persistent runtime volume retains a known sentinel value;
- at least one restored Compose database retains a known sentinel row;
- DB Studio connects to a restored supported database and remains read-only until write mode is explicitly enabled;
- Caddy reconciliation completes and existing routes remain reachable;
- `scripts/verify-production.sh` passes.

The acceptance report may include project identifiers, timing, status, and checksums. It must not include passwords, tokens, cookies, `.env` contents, decrypted project variables, or database credentials.

## Failure handling

Any missing mandatory record, checksum mismatch, unexpected in-use target volume, PostgreSQL restore failure, unreadable encrypted value, missing persistent sentinel, or failed route is a `FAIL`, not a caveat. Destroy the disposable target VM after collecting secret-safe diagnostics, fix the repository-side defect, create a fresh VM, and repeat the drill from the beginning.
