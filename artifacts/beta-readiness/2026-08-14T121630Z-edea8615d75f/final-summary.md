# Final Beta VM Validation Summary

Tested candidate SHA: `edea8615d75f8e032bfb66bf430ab598b05876b2`

VM/environment shape:

- OS/kernel: see `phase-0-preflight/os-kernel.txt`
- CPU: see `phase-0-preflight/cpu.txt`
- RAM: see `phase-0-preflight/ram.txt`
- Disk: see `phase-0-preflight/disk.txt`
- Docker/Compose: see `phase-0-preflight/docker-version.txt` and `phase-0-preflight/docker-compose-version.txt`

Gate matrix:

| Gate | State | Evidence |
| --- | --- | --- |
| Update / release safety | `FAIL` | `phase-1-update-safety/summary.md` |
| Backup / restore | `BLOCKED_ON_VM_EVIDENCE` | `phase-2-backup-restore/summary.md` |
| Many-project performance | `BLOCKED_ON_VM_EVIDENCE` | `phase-3-performance/summary.md` |
| Concurrent-deploy resilience | `BLOCKED_ON_VM_EVIDENCE` | `phase-4-resilience/summary.md` |
| Docker/cache retention | `BLOCKED_ON_VM_EVIDENCE` | `phase-5-docker-retention/summary.md` |
| Create Project runtime contract | `BLOCKED_ON_VM_EVIDENCE` | `phase-6-create-project/summary.md` |
| DB Studio Compose reliability | `BLOCKED_ON_VM_EVIDENCE` | `phase-7-dbstudio/summary.md` |

Defects discovered:

- Candidate release images are missing from GHCR under immutable full-SHA tags for both API and dashboard.

Root cause:

- Release artifact/setup failure: `scripts/update-vm.sh` requires `ghcr.io/nabilrn/mypaas-api:edea8615d75f8e032bfb66bf430ab598b05876b2` and `ghcr.io/nabilrn/mypaas-dashboard:edea8615d75f8e032bfb66bf430ab598b05876b2`, but both manifests returned unavailable.

Focused PRs created:

- None. No product/runtime code defect was isolated; the blocker is missing release artifacts for the candidate SHA.

Final repository CI state:

- PR #110 status check rollup observed from the workstation: Backend tests `SUCCESS`, Frontend checks `SUCCESS`, Deployment script checks `SUCCESS`, Podman compatibility `SUCCESS`, CodeRabbit `SUCCESS`.

Remaining known beta limitations:

- All runtime gates after update safety lack qualifying candidate evidence because the VM could not be updated to the candidate.

Final recommendation:

```text
DO NOT MERGE #110
```
