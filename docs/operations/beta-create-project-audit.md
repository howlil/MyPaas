# Beta Create Project runtime-contract audit

This runbook defines the runtime evidence required for the `core/create-project-runtime-contract` beta gate. The behavioral source of truth remains `docs/ux/create-project-contract.md`.

## Repository-side contract

The mock audit must fail when Create Project becomes enabled in any unresolved state. The enforced scenarios cover:

- slow repository analysis;
- stale analysis after Base Directory changes;
- Dockerfile with missing port;
- Compose required environment values;
- Compose Doctor blocker;
- repository-analysis backend failure;
- repository-analysis timeout;
- static detection;
- registry/GHCR with required port;
- backend project-creation failure reached from a legitimately ready form.

The Vitest readiness contract also locks the fail-closed behavior for stale/incomplete source validation and all busy analysis states.

Run the deterministic mock audit with:

```bash
cd frontend
pnpm audit:create:mock
```

The Playwright harness captures screenshots, ARIA snapshots, visible controls, console events, HAR/network observations, geometry findings, and traces under `frontend/artifacts/create-project-audit/`.

## Production audit

Production mode is intentionally non-destructive. It may inspect/detect a real source and fill normal configuration, but it must not submit a project.

Prepare an authenticated Playwright state, then run:

```bash
cd frontend
pnpm audit:auth
MYPAAS_AUDIT_BASE_URL=https://<mypaas-domain> pnpm audit:create:prod
```

For beta evidence, repeat the production audit with controlled repositories/images that exercise these source/runtime classes and preserve each artifact directory before the next run overwrites the audit root:

- static Git repository;
- Dockerfile Git repository;
- Compose Git repository with an app plus database;
- Git repository whose deployable application is in a subdirectory, using `MYPAAS_AUDIT_SUBDIR_REPO_URL` and `MYPAAS_AUDIT_SUBDIR_PATH`;
- Container Registry/GHCR image using `MYPAAS_AUDIT_IMAGE_REF` and `MYPAAS_AUDIT_REGISTRY_PORT`;
- safely invalid repository for error-state evidence.

Set `MYPAAS_AUDIT_REPO_URL` to the controlled repository for each Git run. Production artifacts must remain secret-safe; authenticated storage state and cookies are not release attachments.

## Acceptance

For each required runtime class, review both the UI checkpoint and the request sequence. The gate fails if any of these occurs:

- Create Project is enabled before repository inspection/detection settles;
- changing branch or Base Directory leaves a previously ready result valid without re-analysis;
- a Dockerfile/registry runtime becomes ready without a port;
- a Compose runtime becomes ready with unresolved required env or a Compose Doctor error;
- a backend failure/timeout leaves stale success state or a ready CTA;
- container-port input loses its typed value after normal input/render cycles;
- Advanced settings are required to recover a normal-flow field that disappeared unexpectedly;
- production audit submits a project;
- console/network/geometry evidence shows an unexplained blocker, crash, request failure, overlap, or horizontal overflow on a required viewport.

The runtime gate remains `BLOCKED_ON_VM_EVIDENCE` until production evidence exists for all required source/runtime classes.
