# Phase 2 Full Backup + Fresh-VM Restore

Status: `BLOCKED_ON_VM_EVIDENCE`

Required evidence not produced:

- Source VM backup with sentinel project state.
- Manifest checksum verification.
- Restore onto a genuinely fresh/disposable VM.
- Post-restore proof for login, project inventory, deployment history, encrypted env usability, static/container/Compose routes, persistent sentinel data, DB sentinel data, DB Studio, Caddy reconciliation, and `scripts/verify-production.sh`.

Blocked reason:

No source MyPaaS VM installation or disposable fresh restore VM is available from this session. Restoring onto this local checkout would violate the runbook and would not count as fresh-VM evidence.
