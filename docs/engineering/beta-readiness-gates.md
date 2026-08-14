# Beta Readiness Gates

This document is the release decision checklist for MyPaas beta. It complements `docs/engineering/beta-readiness-master-plan.md`: the master plan defines the workstreams, while this file defines what evidence is required before a beta claim is allowed.

A green repository is necessary but is not sufficient evidence for runtime reliability. Gates that depend on a real installation remain blocked until the corresponding VM or controlled E2E run has produced evidence.

## Gate states

Use exactly one state for every gate:

- `PASS` — all mandatory evidence exists and the acceptance checks passed.
- `FAIL` — evidence exists and at least one mandatory acceptance check failed.
- `BLOCKED_ON_VM_EVIDENCE` — repository-side implementation or harness exists, but a real VM/E2E execution is still required.
- `NOT_RUN` — the implementation or harness is not yet ready to evaluate.

`BLOCKED_ON_VM_EVIDENCE` is never equivalent to `PASS`.

## Release rule

MyPaas may be called beta only when every mandatory gate below is `PASS`. A beta release is blocked by any `FAIL`, `BLOCKED_ON_VM_EVIDENCE`, or `NOT_RUN` state.

Evidence should be reproducible from a named Git commit and should not contain passwords, tokens, decrypted environment values, cookies, or other secrets.

## Current gate matrix

The initial state is intentionally conservative. Update the state only after linking the evidence produced by the corresponding workstream.

| Gate | State | Repository prerequisite | Mandatory runtime evidence |
| --- | --- | --- | --- |
| Update / release safety | `BLOCKED_ON_VM_EVIDENCE` | SHA-pinned release images, preflight image checks, rollback path, post-update verifier, running build identity | Controlled N→N+1 update and forced failed-update rollback on a VM |
| Backup / restore | `BLOCKED_ON_VM_EVIDENCE` | Backup/restore tooling, manifest/checksum validation, secret-safe reporting | Restore a backup onto a fresh VM and verify control plane plus persistent project data |
| Many-project performance | `BLOCKED_ON_VM_EVIDENCE` | Repeatable 10/25/50-project harness with JSON/Markdown reporting | Run all configured batches on the target VM and capture resource/disk growth and failure rate |
| Concurrent-deploy resilience | `BLOCKED_ON_VM_EVIDENCE` | Concurrent deploy/redeploy/failure harness and consistency checks | Controlled concurrent deployment run including mixed success/failure and route verification |
| Docker/cache retention | `BLOCKED_ON_VM_EVIDENCE` | Safe image/cache retention policy, dry-run path, scheduled cleanup | Run dry-run and cleanup against the target VM and prove running/rollback-critical images remain usable |
| Create Project runtime contract | `BLOCKED_ON_VM_EVIDENCE` | Deterministic mocked contract coverage for critical edge states | Non-destructive production Playwright audit with screenshots, ARIA, network, console, geometry, and trace evidence |
| DB Studio Compose reliability | `BLOCKED_ON_VM_EVIDENCE` | Resolver/credential tests and Compose database smoke fixtures | Connect through MyPaas to MariaDB, MySQL, and PostgreSQL Compose services on the project network |
| Beta documentation / limitations | `PASS` | This checklist plus the master plan and explicit evidence rules | No additional runtime execution; this gate must be revised when release caveats change |

## Evidence contract

Store generated beta-readiness evidence outside source-controlled secrets. A run should have a stable run identifier containing at least the UTC timestamp and tested Git SHA, for example:

```text
artifacts/beta-readiness/2026-08-14T070000Z-455a74e/
```

Each automated harness should emit a machine-readable JSON result. Performance and resilience harnesses should also emit a concise Markdown summary suitable for attaching to a pull request or release decision record.

Every evidence record should include, where applicable:

- tested Git SHA and build/image identity;
- target environment identifier without credentials;
- started/finished timestamps;
- scenario or batch name;
- pass/fail result and threshold used;
- failed requests or state-transition anomalies;
- host CPU, memory, storage, and Docker/cache observations when available;
- explicit `blocked_reason` when a check could not execute.

## Gate checklists

### Update / release safety

- [ ] Target API and dashboard images are immutable SHA tags matching the intended Git commit.
- [ ] Missing target images leave the current checkout and runtime unchanged.
- [ ] Post-update verification checks API readiness and dashboard reachability.
- [ ] Caddy route reconciliation is verified after update.
- [ ] At least one pre-existing project route remains reachable after update.
- [ ] Forced post-update failure restores the previous checkout and runtime images.
- [ ] The running build SHA/version is visible through the owner-facing API or dashboard.
- [ ] VM evidence links the before SHA, target SHA, rollback drill, and final running SHA.

### Backup / restore

- [ ] Backup includes the MyPaas control-plane PostgreSQL database.
- [ ] Required production configuration is captured with restrictive permissions and never printed in reports.
- [ ] Static artifacts under `/var/lib/mypaas/static` are captured.
- [ ] Managed persistent Docker volumes are captured.
- [ ] Backup manifest contains checksums and no secret values.
- [ ] Restore tooling validates the manifest before mutating a target.
- [ ] Fresh-VM restore verifies login, project inventory, deployment history, routes, encrypted env readability, persistent project data, and DB Studio.
- [ ] Restore verification emits a machine-readable report.

### Many-project performance

- [ ] Static, Dockerfile, and Compose app+database fixtures are available.
- [ ] Harness can execute 10, 25, and 50-project batches from configuration.
- [ ] Report includes create/deploy/build timing, failure rate, and route verification.
- [ ] Report captures CPU, memory, storage growth, and Docker image/cache growth when host telemetry is available.
- [ ] Disk growth is attributed to images, BuildKit cache, volumes, logs, or MyPaas artifacts where the host probe supports it.
- [ ] Thresholds are recorded in the report rather than inferred after the run.
- [ ] Any project-count limit encountered is reported as an explicit capacity result.

### Concurrent-deploy resilience

- [ ] Harness can trigger concurrent deploy/redeploy requests.
- [ ] Harness can mix intentionally failing work with successful deployments.
- [ ] Webhook-burst mode is available only as an explicitly enabled destructive integration scenario.
- [ ] Final project/deployment state is checked after all workers settle.
- [ ] No duplicate allocated host ports are observed.
- [ ] No deployment remains stuck in a non-terminal state beyond the configured timeout.
- [ ] Existing unrelated project routes remain reachable.
- [ ] Failed deployments do not replace a previously healthy runtime.

### Docker/cache retention

- [ ] Managed-image cleanup is scoped to MyPaas-managed images.
- [ ] Running images are never deleted by the cleanup path.
- [ ] A recent-image/rollback retention window is explicit.
- [ ] BuildKit cache has a scheduled retention path.
- [ ] The same cleanup can be invoked manually.
- [ ] Dry-run output shows what would be inspected/reclaimed without deleting data.
- [ ] Host evidence records images, build cache, volumes, logs, and MyPaas artifact usage before and after cleanup where supported.

### Create Project runtime contract

Use `docs/ux/create-project-contract.md` as the behavioral source of truth.

- [ ] Mock audit covers static, Dockerfile, Compose, registry/GHCR, required env, missing port, Compose Doctor blockers, base-directory handling, slow analysis, backend failure/timeout, stale re-analysis, and creation failure.
- [ ] Create cannot become ready while analysis is stale, scheduled, incomplete, or in flight.
- [ ] Production audit is non-destructive by default.
- [ ] Production evidence includes screenshots, ARIA/accessibility representation, console, network, geometry, and Playwright trace data.
- [ ] Git, registry/GHCR, Compose, static, Dockerfile, and subdirectory flows have evidence.

### DB Studio Compose reliability

- [ ] Resolver tests cover project env and Compose database-service env fallback.
- [ ] MariaDB, MySQL, and PostgreSQL Compose fixtures exist.
- [ ] Missing/incomplete credentials produce actionable errors without exposing secrets.
- [ ] DB Studio resolves the real Compose service/network path instead of inventing fallback network names.
- [ ] Read-only access remains the default.
- [ ] Write mode requires an explicit, expiring write session.
- [ ] Controlled smoke evidence proves a connection to all three supported database fixtures.

## Known beta caveats

Keep this section intentionally short and update it before release. Caveats are acceptable only when they do not invalidate a mandatory gate.

- MyPaas remains a small self-hosted platform rather than a multi-tenant managed cloud service.
- Performance limits are installation-specific; publish the tested VM shape alongside 10/25/50-project evidence.
- Host-level verification and destructive resilience drills must run in a controlled test installation, not against unrelated production workloads.
- Production Create Project audits remain non-destructive unless a scenario is explicitly promoted to a controlled integration test.

## Evidence update procedure

1. Run the repository checks required by the workstream.
2. Run the VM/E2E scenario when the gate requires it.
3. Attach or link the generated JSON/Markdown evidence to the workstream PR or release record.
4. Change a gate to `PASS` only when every mandatory checklist item is satisfied.
5. Record any failure as `FAIL`; do not hide it behind a caveat.
6. Re-run affected gates whenever update, backup, deployment scheduling, routing, persistence, or database-connection behavior materially changes.
