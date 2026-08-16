# Beta Readiness Gates

This document is the release decision checklist for MyPaas beta. It complements `docs/engineering/beta-readiness-master-plan.md`: the master plan defines the workstreams, while this file records the evidence-backed release state.

Repository CI is necessary but is not sufficient evidence for runtime reliability. Runtime gates are marked `PASS` only when the corresponding VM or controlled E2E qualification evidence exists.

## Qualified runtime candidate

The runtime candidate qualified for beta readiness is:

```text
ddc26c9a0f877fc5dd4133d6559c5f36123d6a31
```

The gate reconciliation in this document is documentation-only. A later docs-only merge SHA does not replace the runtime qualification candidate unless runtime-affecting code changes are introduced.

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

| Gate | State | Tested SHA | Qualification basis |
| --- | --- | --- | --- |
| Update / release safety | `PASS` | `edea8615d75f8e032bfb66bf430ab598b05876b2` | Controlled update, missing-image safety, rollback, health, build identity, and route verification. Carried forward after blast-radius review. |
| Backup / restore | `PASS` | `de63a285c621125959d0612e4eacb7397c887a0a` | Fresh-VM restore of control-plane data, configuration, static artifacts, managed volumes, routes, encrypted env usability, and DB Studio. Carried forward after blast-radius review. |
| Many-project performance | `PASS` | `8c724026841ea31ed5b1bd39d5e7f16b1bd0a4b1` | 10/25/50-project qualification with static, Dockerfile, and Compose fixtures; 50/50 successful at the largest tier. Carried forward after focused redeploy-only fix review. |
| Concurrent-deploy resilience | `PASS` | `ddc26c9a0f877fc5dd4133d6559c5f36123d6a31` | Concurrent create/deploy/redeploy, intentional failure, protected route, webhook burst, read paths, and final port/runtime consistency. |
| Docker/cache retention | `PASS` | `ddc26c9a0f877fc5dd4133d6559c5f36123d6a31` | Managed-image dry-run/apply, running and rollback image protection, unrelated image safety, persistent-volume safety, scheduled/manual retention, and post-cleanup deploy/rollback. |
| Create Project runtime contract | `PASS` | `ddc26c9a0f877fc5dd4133d6559c5f36123d6a31` | Deterministic mock audit plus authenticated non-destructive production audit across static, Dockerfile, Compose, subdirectory, registry/GHCR, and invalid-repository flows. |
| DB Studio Compose reliability | `PASS` | `ddc26c9a0f877fc5dd4133d6559c5f36123d6a31` | PostgreSQL, MySQL, and MariaDB Compose smoke; real network/service resolution; fallback/precedence; secret-safe metadata; read-only default and expiring write sessions. |
| Beta documentation / limitations | `PASS` | documentation gate | This checklist, master plan, evidence provenance, historical failures, and explicit beta caveats are recorded. |

All eight mandatory beta-readiness gates are evidence-derived `PASS` with zero open mandatory blockers.

## Evidence index

Durable evidence is stored on the primary reference VM (`172.105.118.30`) and is intentionally not committed to the repository because the archives contain operational test artifacts.

| Gate | Evidence reference |
| --- | --- |
| Update / release safety | `/root/MyPaas/artifacts/beta-readiness/2026-08-14T123323Z-edea8615d75f/phase-1-update-safety/` |
| Backup / restore | `/root/beta-readiness-evidence/de63a285/phase-2-backup-restore/` and `/root/full-final-de63a285.tar.gz` |
| Many-project performance | `/root/beta-readiness-evidence/8c724026/phase-3-many-project-performance/phase3_evidence_8c724026.tar.gz` |
| Concurrent-deploy resilience | `/root/beta-readiness-evidence/ddc26c9a/phase-4-resilience-concurrent-deploys/phase4_official_evidence_ddc26c9a.tar.gz` |
| Docker/cache retention | `/root/beta-readiness-evidence/ddc26c9a/phase-5-docker-cache-retention/phase5_official_evidence_ddc26c9a.tar.gz` |
| Create Project runtime contract | `/root/beta-readiness-evidence/ddc26c9a/phase-6-create-project-runtime-contract/phase6_create_project_evidence_ddc26c9a.tar.gz` |
| DB Studio Compose reliability | `/root/beta-readiness-evidence/ddc26c9a/phase-7-dbstudio-compose-reliability/phase7_dbstudio_evidence_ddc26c9a.tar.gz` |
| Final consolidation | `/root/beta-readiness-evidence/ddc26c9a/final-consolidation/mypaas_beta_readiness_final_ddc26c9a.tar.gz` |

## Carry-forward provenance

Not every gate was re-executed on the final runtime SHA. Carry-forward is allowed only when later changes do not materially intersect the gate's mandatory acceptance criteria and the changed runtime path is requalified by a later gate.

- **Update / release safety** — carried forward from `edea8615d75f8e032bfb66bf430ab598b05876b2`. Later PRs #125 and #126 changed Compose/deployment port behavior, not the update/rollback scripts, image preflight, build identity, or Caddy reconciliation path covered by this gate.
- **Backup / restore** — carried forward from `de63a285c621125959d0612e4eacb7397c887a0a`. Later PRs #125 and #126 did not change backup/restore tooling or archive semantics; restored runtime behavior was subsequently exercised by later deployment/DB Studio gates.
- **Many-project performance** — carried forward from `8c724026841ea31ed5b1bd39d5e7f16b1bd0a4b1`. PR #126 changed side-by-side Dockerfile replacement allocation, not initial multi-project creation capacity. The changed path was directly requalified by the final concurrent-redeploy gate on `ddc26c9a`.

## Historical failures

Historical failures remain immutable engineering evidence even after their defects are fixed.

- `de63a285c621125959d0612e4eacb7397c887a0a` — **Phase 3 FAIL**: Compose empty-health handling caused deployment timeout and port divergence. Resolved by PR #125 and requalified on successor candidates.
- `8c724026841ea31ed5b1bd39d5e7f16b1bd0a4b1` — **Phase 4 FAIL**: Dockerfile side-by-side redeploy attempted to reuse the active runtime port and failed with a self-collision. Resolved by PR #126; official Phase 4 subsequently passed on `ddc26c9a`.
- `edea8615d75f8e032bfb66bf430ab598b05876b2` — **Phase 7 FAIL**: the old DB Studio smoke passed a raw immutable commit SHA as a project branch. Fixture-ref resolution now maps an immutable SHA to a real remote branch and verifies the branch head before destructive execution.

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

- [x] Target API and dashboard images are immutable SHA tags matching the intended Git commit.
- [x] Missing target images leave the current checkout and runtime unchanged.
- [x] Post-update verification checks API readiness and dashboard reachability.
- [x] Caddy route reconciliation is verified after update.
- [x] At least one pre-existing project route remains reachable after update.
- [x] Forced post-update failure restores the previous checkout and runtime images.
- [x] The running build SHA/version is visible through the owner-facing API or dashboard.
- [x] VM evidence links the before SHA, target SHA, rollback drill, and final running SHA.

### Backup / restore

- [x] Backup includes the MyPaas control-plane PostgreSQL database.
- [x] Required production configuration is captured with restrictive permissions and never printed in reports.
- [x] Static artifacts under `/var/lib/mypaas/static` are captured.
- [x] Managed persistent Docker volumes are captured.
- [x] Backup manifest contains checksums and no secret values.
- [x] Restore tooling validates the manifest before mutating a target.
- [x] Fresh-VM restore verifies login, project inventory, deployment history, routes, encrypted env readability, persistent project data, and DB Studio.
- [x] Restore verification emits a machine-readable report.

### Many-project performance

- [x] Static, Dockerfile, and Compose app+database fixtures are available.
- [x] Harness can execute 10, 25, and 50-project batches from configuration.
- [x] Report includes create/deploy/build timing, failure rate, and route verification.
- [x] Report captures CPU, memory, storage growth, and Docker image/cache growth when host telemetry is available.
- [x] Disk growth is attributed to images, BuildKit cache, volumes, logs, or MyPaas artifacts where the host probe supports it.
- [x] Thresholds are recorded in the report rather than inferred after the run.
- [x] Any project-count limit encountered is reported as an explicit capacity result.

### Concurrent-deploy resilience

- [x] Harness can trigger concurrent deploy/redeploy requests.
- [x] Harness can mix intentionally failing work with successful deployments.
- [x] Webhook-burst mode is available only as an explicitly enabled destructive integration scenario.
- [x] Final project/deployment state is checked after all workers settle.
- [x] No duplicate allocated host ports are observed.
- [x] No deployment remains stuck in a non-terminal state beyond the configured timeout.
- [x] Existing unrelated project routes remain reachable.
- [x] Failed deployments do not replace a previously healthy runtime.

### Docker/cache retention

- [x] Managed-image cleanup is scoped to MyPaas-managed images.
- [x] Running images are never deleted by the cleanup path.
- [x] A recent-image/rollback retention window is explicit.
- [x] BuildKit cache has a scheduled retention path.
- [x] The same cleanup can be invoked manually.
- [x] Dry-run output shows what would be inspected/reclaimed without deleting data.
- [x] Host evidence records images, build cache, volumes, logs, and MyPaas artifact usage before and after cleanup where supported.

### Create Project runtime contract

Use `docs/ux/create-project-contract.md` as the behavioral source of truth.

- [x] Mock audit covers static, Dockerfile, Compose, registry/GHCR, required env, missing port, Compose Doctor blockers, base-directory handling, slow analysis, backend failure/timeout, stale re-analysis, and creation failure.
- [x] Create cannot become ready while analysis is stale, scheduled, incomplete, or in flight.
- [x] Production audit is non-destructive by default.
- [x] Production evidence includes screenshots, ARIA/accessibility representation, console, network, geometry, and Playwright trace data.
- [x] Git, registry/GHCR, Compose, static, Dockerfile, and subdirectory flows have evidence.

### DB Studio Compose reliability

- [x] Resolver tests cover project env and Compose database-service env fallback.
- [x] MariaDB, MySQL, and PostgreSQL Compose fixtures exist.
- [x] Missing/incomplete credentials produce actionable errors without exposing secrets.
- [x] DB Studio resolves the real Compose service/network path instead of inventing fallback network names.
- [x] Read-only access remains the default.
- [x] Write mode requires an explicit, expiring write session.
- [x] Controlled smoke evidence proves a connection to all three supported database fixtures.

## Known beta caveats

These caveats are non-blocking because they do not invalidate a mandatory beta-readiness gate.

- MyPaas remains a small, single-node self-hosted platform rather than a multi-tenant managed cloud service.
- Performance limits are installation-specific. The qualified 50-project run used a 4-vCPU, approximately 8-GiB VM; larger installations should re-establish their own capacity envelope.
- Host-level destructive drills such as restore, resilience, and retention qualification belong on controlled test installations rather than unrelated production workloads.
- Production Create Project audits remain non-destructive unless a scenario is explicitly promoted to a controlled integration test.
- The dashboard does not yet surface a disk-pressure warning. Manual and scheduled managed-image/cache retention remains operational.
- The Phase 4 Python `urllib` harness can observe `http.client.RemoteDisconnected` during high-rate webhook activity when a keep-alive connection closes; qualification evidence showed coherent server-side terminal processing and no product-state corruption.
- Phase 6 audit artifact directory names retain a historical `firefox` label even though the qualified Playwright engine was Chromium. This is an evidence-labeling anomaly, not Firefox coverage.

## Release decision

Final consolidation for runtime candidate `ddc26c9a0f877fc5dd4133d6559c5f36123d6a31` concluded:

```text
BETA-READY — SAFE TO PREPARE RELEASE
```

This decision means the runtime candidate cleared the mandatory beta-readiness gates. Release publication still requires normal release preparation, version/tag selection, release notes, immutable image publication, and provenance verification.

## Evidence update procedure

1. Run the repository checks required by the workstream.
2. Run the VM/E2E scenario when the gate requires it.
3. Preserve the generated JSON/Markdown evidence and durable archive outside source-controlled secrets.
4. Change a gate to `PASS` only when every mandatory checklist item is satisfied.
5. Record any failure as `FAIL`; do not hide it behind a caveat.
6. Preserve historical failed candidates after fixes; never rewrite them as PASS.
7. Re-run affected gates whenever update, backup, deployment scheduling, routing, persistence, or database-connection behavior materially changes.
8. For carried-forward gates, record the later change's blast radius and why it does or does not invalidate the prior evidence.
