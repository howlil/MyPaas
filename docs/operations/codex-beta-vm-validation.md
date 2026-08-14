# Codex Handoff — Final Beta VM Validation

Use this document as the execution prompt for the final MyPaaS beta-readiness validation.

Repository:

- `https://github.com/nabilrn/MyPaas`
- integration PR: `#110` — `test: integrate beta readiness candidate`
- target branch: `test/beta-readiness-candidate`
- base branch: `main`
- repository-integration baseline before this handoff document was added: `688712704c54a249447e2c0ae864c94e492b93ca`

Do **not** trust the baseline SHA as the current candidate SHA after this document is merged. Resolve the current `origin/test/beta-readiness-candidate` first and verify that PR #110 points to the same head.

## Non-negotiable rules

- Do **not** merge PR #110 into `main` automatically.
- Do **not** redesign the UI.
- Do **not** rewrite already-completed beta-readiness work unless runtime evidence exposes a real defect.
- Do **not** claim a beta gate `PASS` without reproducible runtime evidence.
- Optimize for finding real defects, not for making every result green.
- Preserve failing evidence before changing code.
- Keep fixes focused and follow `docs/engineering/branching.md`.
- Never commit passwords, API tokens, cookies, decrypted project environment values, SSH private keys, or other secrets.

## Repository-side state already completed

The following beta-readiness workstreams are integrated in the candidate:

- #102 `docs/beta-readiness-gates`
- #103 `test/perf-many-projects`
- #104 `test/resilience-concurrent-deploys`
- #105 `infra/docker-cache-retention`
- #106 `core/update-release-safety`
- #107 `core/backup-restore`
- #108 `core/create-project-runtime-contract`
- #109 `core/dbstudio-compose-reliability`

The combined candidate already passed repository-level validation before this handoff document was added:

- GitHub CI #338: PASS
- Podman compatibility #241: PASS
- backend tests: PASS
- Go race detector: PASS
- frontend unit tests: PASS
- frontend type/Svelte checks: PASS
- frontend production build: PASS
- shell syntax: PASS
- script regression tests: PASS
- benchmark harness unit tests: PASS
- production Compose rendering: PASS
- rootful Podman compatibility smoke: PASS

Repository CI is necessary but is not runtime evidence. The remaining release gates intentionally remain `BLOCKED_ON_VM_EVIDENCE` until the VM/E2E drills below are executed.

## Read first

Before doing any work, read:

- `AGENTS.md` and every nested `AGENTS.md` that applies to files you touch
- `docs/engineering/branching.md`
- `docs/engineering/beta-readiness-master-plan.md`
- `docs/engineering/beta-readiness-gates.md`
- `docs/operations/beta-backup-restore-drill.md`
- `docs/operations/beta-create-project-audit.md`
- this file

## Primary objective

Produce reproducible VM/E2E evidence for all remaining beta-readiness gates:

1. update / release safety
2. backup / fresh-VM restore
3. 10 / 25 / 50 project performance
4. concurrent deployment resilience
5. Docker / BuildKit retention
6. authenticated Create Project production audit
7. DB Studio PostgreSQL / MySQL / MariaDB Compose smoke

Execute them sequentially and preserve evidence after every phase.

If a phase fails, determine whether the failure is:

- environment/setup failure;
- harness defect;
- product/runtime defect.

Only patch actual defects. Do not weaken acceptance criteria merely to obtain a green result.

---

# Phase 0 — Pre-flight

Before mutating the VM:

1. Confirm repository state:

```bash
git status
git branch --show-current
git rev-parse HEAD
git fetch origin
```

2. Resolve the current candidate:

```bash
git rev-parse origin/test/beta-readiness-candidate
```

3. Confirm PR #110 is still open, targets `main`, is not merged, and its head matches the candidate SHA.

4. Verify the candidate still contains the integrated #102–#109 workstreams.

5. Record VM baseline:

- OS and kernel
- CPU count/model where practical
- RAM
- disk size/free space
- Docker/Podman version
- current MyPaaS checkout SHA
- running API image
- running dashboard image
- API `MYPAAS_BUILD_SHA`
- `docker ps`
- `docker system df`
- `docker builder du`
- filesystem usage
- memory/load
- production Compose status
- Caddy status
- at least one existing deployed project route when available

6. Create one evidence root:

```text
artifacts/beta-readiness/<UTC_TIMESTAMP>-<CANDIDATE_SHA>/
```

Every phase must write sanitized raw evidence and a concise summary beneath this directory.

Never store credentials or decrypted secrets in evidence.

---

# Phase 1 — Update / release safety

Relevant files include:

- `scripts/update-vm.sh`
- `scripts/deploy-to-vm.sh`
- `scripts/verify-production.sh`
- `scripts/update_release_safety_test.py`

## 1A. Successful N → N+1 update

Record before-state:

- checkout SHA
- API image identity
- dashboard image identity
- API `MYPAAS_BUILD_SHA`
- API health/readiness
- dashboard reachability
- Caddy state
- one pre-existing project route

Perform a controlled update using exact immutable Git-SHA image tags.

After update prove:

- checkout equals target SHA
- API image tag equals target SHA
- dashboard image tag equals target SHA
- `MYPAAS_BUILD_SHA` equals target SHA
- `/health` passes
- `/ready` passes
- dashboard works
- Caddy works
- the pre-existing project route still works

## 1B. Missing-image safety

Attempt a controlled update to a deliberately unavailable image SHA/tag.

Prove:

- checkout did not advance to the bad target
- API runtime did not change
- dashboard runtime did not change
- existing project route remains reachable

## 1C. Forced post-update verification failure

Cause a controlled post-update verification failure after target selection.

Prove automatic rollback restores:

- previous checkout SHA
- previous API runtime image
- previous dashboard runtime image
- previous `MYPAAS_BUILD_SHA`
- API health/readiness
- dashboard
- Caddy
- existing project route

Pay special attention to local `rollback-<sha>` images. A previous updater defect attempted to pull these local rollback tags from GHCR. Verify the corrected local-image rollback path actually works.

Store at minimum:

- `update-success.json` / `.md`
- `missing-image.json` / `.md`
- `rollback-drill.json` / `.md`
- sanitized relevant command output

Do not mark the update gate `PASS` unless all mandatory checks pass.

---

# Phase 2 — Full backup + fresh-VM restore

Use:

```text
scripts/backup-restore.py
```

Follow:

```text
docs/operations/beta-backup-restore-drill.md
```

This must be a real disaster-recovery exercise. Restoring back onto the same installation is not sufficient fresh-VM evidence.

Before backup, create or select sentinel state that can be checked after restore:

- known project inventory
- deployment history
- encrypted project env entry
- static artifact
- persistent Docker/Compose data
- preferably a Compose database sentinel row

The source bundle must include:

- control-plane PostgreSQL
- production configuration
- `/var/lib/mypaas/static`
- `/var/lib/mypaas/compose`
- MyPaaS-managed project volumes

Use quiescing where the tool requires it. Verify manifest checksums before restore mutation.

Restore onto a genuinely fresh/disposable VM when infrastructure permits, using the exact backup source Git SHA unless the runbook explicitly says otherwise.

After restore prove:

- owner login works
- project inventory survives
- deployment history survives
- encrypted env remains usable
- reports do not expose secret values
- static route works
- Dockerfile/container route works
- Compose route works
- persistent sentinel data survives
- DB sentinel survives where configured
- DB Studio works
- Caddy reconciliation works
- `scripts/verify-production.sh` passes

Do not perform manual database surgery to make the restore pass. If manual repair is required, preserve evidence and classify the gate `FAIL` until a proper fix is implemented and the drill is rerun.

Store the manifest, checksum verification, `restore-report.json`, a Markdown summary, and sanitized before/after inventory.

---

# Phase 3 — 10 / 25 / 50 project performance

Use:

```text
benchmarks/beta_perf.py
```

Fixtures:

```text
benchmarks/fixtures/beta/
```

Use the current candidate SHA as both tested Git SHA and fixture ref.

Run stages:

- 10 projects
- 25 projects
- 50 projects

Capture for each stage:

- project creation duration
- deployment duration
- total duration
- p50
- p95
- max
- failure rate
- duplicate allocated ports
- host CPU
- host memory
- disk growth
- Docker image growth
- BuildKit cache growth
- volume growth
- logs/artifact growth where measurable

Manually verify at least one successful deployment of each fixture type:

- static
- Dockerfile
- Compose app + database

Do not hide a VM capacity boundary. If the 50-project stage exceeds the tested VM capacity, report the actual limit as engineering evidence.

Do not raise thresholds merely to turn a failure into a pass.

Preserve the generated `report.json` and `report.md`.

---

# Phase 4 — Concurrent deployment resilience

Use:

```text
benchmarks/beta_resilience.py
```

Run a controlled scenario containing:

- concurrent project creation
- concurrent deploy
- concurrent redeploy
- successful and deliberately failing builds together
- logs reads during active deployments
- metrics reads during active deployments
- DB Studio read path during active deployments
- a protected unrelated existing route
- optional signed webhook burst when the environment is appropriate

Prove:

- no duplicate host-port allocations
- no deployments remain stuck past timeout
- workers settle into terminal states
- failed builds do not destroy the previous healthy runtime
- successful project routes remain reachable
- unrelated protected route remains reachable
- Caddy state is coherent after workers settle
- final project/runtime/deployment state is consistent

If webhook burst is enabled:

- use only a controlled project/webhook
- sign requests correctly
- never expose the webhook secret in evidence

Preserve JSON and Markdown reports.

---

# Phase 5 — Docker / BuildKit retention

Use:

```text
scripts/docker-retention.sh
```

Run dry-run first.

Record before-state:

- `docker system df`
- running image references
- MyPaaS-managed images
- BuildKit cache
- volume usage
- MyPaaS artifact usage
- updater lock state

Then execute controlled cleanup using `--apply`.

Prove:

- running images remain usable
- active project routes remain healthy
- rollback/recent image policy is preserved
- old managed images follow retention policy
- BuildKit cache is reclaimed as expected
- unrelated host images are not unexpectedly removed
- updater lock blocks cleanup while an update is active

After cleanup, redeploy a controlled project and verify its route/runtime.

Capture exact reclaimed disk amounts where possible.

Do **not** substitute broad host cleanup such as:

```bash
docker system prune -a
```

Only test the implemented MyPaaS retention path.

---

# Phase 6 — Create Project authenticated production audit

Do not redesign Create Project.

Use the existing production Playwright audit and:

```text
docs/operations/beta-create-project-audit.md
```

Run against an authenticated controlled installation.

Cover:

1. static Git repository
2. Dockerfile repository
3. Compose app + database
4. nested Base Directory repository
5. GHCR / registry deployment
6. invalid/broken repository

Also validate the behavioral invariants represented by the mock audit:

- slow analysis
- stale analysis
- Base Directory changes invalidate prior analysis
- missing Dockerfile port
- registry flow requiring a port
- required Compose env
- Compose Doctor blocker
- backend 500
- timeout
- project creation failure

Production audit remains non-destructive where the runbook requires it.

Preserve, where supported:

- screenshots
- ARIA/accessibility representation
- network evidence
- browser console
- geometry/layout evidence
- Playwright trace

Critical invariant:

> Create must never become ready while source inspection is stale, incomplete, scheduled, or still running.

Treat any stale-analysis readiness violation as a beta blocker.

---

# Phase 7 — DB Studio Compose reliability

Use:

```text
benchmarks/dbstudio_compose_smoke.py
```

Fixtures:

- `benchmarks/fixtures/dbstudio/postgres/`
- `benchmarks/fixtures/dbstudio/mysql/`
- `benchmarks/fixtures/dbstudio/mariadb/`

Run all three engines through actual MyPaaS deployment.

For each engine prove:

- project deploys
- Compose DB service becomes healthy
- DB Studio detects configuration
- expected driver is selected
- actual project Compose network/service path is used
- connection succeeds
- schema read path succeeds
- `writeAccess` is absent/null by default
- public connection metadata does not expose credential values

Where practical, verify project-env precedence over Compose-service fallback.

Write mode must remain:

- explicit
- time-limited
- disabled by default

Do not enable write mode merely to make the read-only smoke pass.

---

# Gate update policy

After all phases, update beta readiness only from evidence.

Use:

```text
docs/engineering/beta-readiness-gates.md
```

Allowed states:

- `PASS`
- `FAIL`
- `BLOCKED_ON_VM_EVIDENCE`
- `NOT_RUN`

Interpret them strictly:

- `PASS`: every mandatory acceptance check has reproducible evidence and passed.
- `FAIL`: evidence exists and a mandatory acceptance criterion failed.
- `BLOCKED_ON_VM_EVIDENCE`: a required runtime execution genuinely could not be performed.
- `NOT_RUN`: the implementation/harness itself is not ready to evaluate.

Never translate “CI is green” into “runtime gate PASS”.

---

# Fix policy for runtime defects

When runtime validation reveals a genuine defect:

1. Preserve failing evidence.
2. Reproduce it deterministically.
3. Identify root cause before patching.
4. Create a focused branch using `docs/engineering/branching.md`.
5. Add regression coverage first or alongside the fix.
6. Keep the implementation minimal and focused.
7. Run relevant repository checks.
8. Open the fix PR against `test/beta-readiness-candidate`, **not** `main`.
9. Wait for CI and Podman checks where applicable.
10. Merge the focused fix into the candidate only after it is green.
11. Rerun the failed VM gate.
12. Update evidence and gate status only after the rerun.

Examples of acceptable focused branch names:

- `core/backup-compose-volume-restore`
- `core/update-rollback-verification`
- `core/dbstudio-compose-network-fix`
- `infra/docker-retention-rollback-images`
- `test/resilience-route-consistency`

Do not perform speculative refactoring, unrelated cleanup, or architectural rewrites during this validation.

---

# Final deliverable

At completion, report:

1. exact tested candidate SHA;
2. VM/environment shape:
   - OS
   - CPU
   - RAM
   - disk
   - Docker/Podman version;
3. gate matrix for:
   - update safety
   - backup/restore
   - performance
   - resilience
   - Docker retention
   - Create Project
   - DB Studio;
4. `PASS` / `FAIL` / `BLOCKED_ON_VM_EVIDENCE` / `NOT_RUN` for each;
5. evidence path/link for every gate;
6. every defect discovered;
7. root cause of every defect;
8. focused PRs created to fix defects;
9. final repository CI state;
10. remaining known beta limitations;
11. final recommendation exactly as one of:

```text
SAFE TO MERGE #110
```

or

```text
DO NOT MERGE #110
```

Even if every gate passes, **do not merge PR #110 automatically**. Stop at the recommendation and wait for owner approval.
