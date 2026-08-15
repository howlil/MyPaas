# Final Beta VM Validation Summary

Tested candidate SHA: `edea8615d75f8e032bfb66bf430ab598b05876b2`

VM/environment shape:

- OS/kernel: Debian GNU/Linux 13, kernel `6.12.88+deb13-amd64`
- CPU: 2 vCPU, AMD EPYC 7713
- RAM: 3.8 GiB
- Disk: `/dev/sda` 79G total, 63G available at preflight
- Docker: Engine 29.7.2
- Docker Compose: v5.4.0

Gate matrix:

| Gate | State | Evidence |
| --- | --- | --- |
| Update / release safety | `FAIL` | `phase-1-update-safety/summary.md` |
| Backup / restore | `BLOCKED_ON_VM_EVIDENCE` | `phase-2-backup-restore/summary.md` |
| Many-project performance | `FAIL` | `phase-3-performance/summary.md` |
| Concurrent-deploy resilience | `FAIL` | `phase-4-resilience/summary.md` |
| Docker/cache retention | `PASS` | `phase-5-docker-retention/summary.md` |
| Create Project runtime contract | `BLOCKED_ON_VM_EVIDENCE` | `phase-6-create-project/summary.md` |
| DB Studio Compose reliability | `FAIL` | `phase-7-dbstudio/summary.md` |

Defects discovered:

1. Failed-update rollback from the VM's initial updater attempts to pull `rollback-<sha>` from GHCR instead of using local rollback images.
2. The beta performance, resilience, and DB Studio harnesses cannot satisfy the handoff's exact-SHA fixture requirement because they pass the SHA as an API project `branch`.

Root causes:

1. The initial installed updater at `b94361be993963c0ad04a5c3a76130bc8d8ef8ee` does not contain the candidate's local rollback-image fix; during a failed upgrade, the old running script controls rollback.
2. Harness/API contract mismatch: project creation accepts cloneable branch names, while the handoff requires fixture ref to be the exact candidate commit SHA.

Focused PRs created:

- None. Fixes were not implemented in this validation run. The updater compatibility issue likely needs a bridge/update strategy, because candidate code cannot change the already-running updater from `b94361be993963c0ad04a5c3a76130bc8d8ef8ee`. The harness/API exact-SHA issue needs a focused test harness/API contract fix before rerunning Phases 3, 4, and 7.

Final repository CI state:

- PR #110 status check rollup was previously observed as successful for Backend tests, Frontend checks, Deployment script checks, Podman compatibility, and CodeRabbit.

Remaining known beta limitations:

- No qualifying fresh-VM restore evidence.
- No qualifying Create Project authenticated production audit evidence.
- Performance/resilience/DB Studio runtime capacity could not be evaluated because exact-SHA fixture project creation fails.

Final recommendation:

```text
DO NOT MERGE #110
```
