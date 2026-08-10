---
name: mypaas-dashboard-state-audit
description: Audit MyPaas SvelteKit dashboard state management, async state, and state-driven UX. Use when reviewing or improving frontend pages, stores, forms, realtime views, deployment flows, environment variables, settings, metrics, logs, tables, empty/loading/error states, or any MyPaas dashboard UX issue caused by inconsistent state.
---

# MyPaas Dashboard State Audit

Use this skill to find state bugs before changing visuals. MyPaas is an operational PaaS dashboard; every screen must make source of truth, pending work, stale data, failures, and next actions obvious.

## Inputs To Inspect

- Route pages in `frontend/src/routes/**/+page.svelte` and nested project layouts.
- Shared state in `frontend/src/lib/stores/`.
- API contracts in `frontend/src/lib/api/` and `frontend/src/lib/types.ts`.
- Shared controls in `frontend/src/lib/components/`, especially `ActionButton`, `SecretField`, `TableShell`, `Pagination`, `DeployControlPanel`, `ErrorState`, and `EmptyState`.

Prefer codebase-memory graph tools for discovering components/functions. Use file reads for Svelte markup and local transient state because graph snippets are not enough for UX state audits.

## Audit Method

1. Map the user workflow: entry state, data load, edit, validation, submit, success, error, cancel, navigation away.
2. Identify every state owner:
   - URL or route params
   - API-loaded server data
   - local draft state
   - derived reactive state
   - stores
   - timers, EventSource, intervals, and background polling
3. Check source-of-truth boundaries:
   - Server data must not be silently overwritten by stale local drafts.
   - Draft state must have explicit dirty/discard semantics.
   - Derived state must not duplicate mutable state unless there is a reset path.
   - Route changes must reset page-local state keyed by project id or query params.
4. Check async state:
   - Each async action has a dedicated pending flag.
   - Pending flags reset in `finally`.
   - Double submits are blocked.
   - Race-prone requests use request ids, abort controllers, or key checks.
   - Background reloads do not hide stale data as fresh data.
5. Check UX state:
   - Loading, empty, error, partial, dirty, disabled, and success states are visible.
   - Disabled controls explain why when the reason is not obvious.
   - Destructive and secret actions have independent pending states.
   - Mobile layouts preserve action labels and do not depend on hover-only affordances.

## MyPaas-Specific Rules

- Project creation must be source-first, then repository validation/detection, then runtime/resources/env review, then create.
- Git project creation must validate the current repo URL, branch, and base directory before submit.
- Registry project creation must not require repository validation.
- Static projects must clearly force port 80 and disable shared database options.
- Compose mode must surface main service, compose file, overrides, profiles, workdir, required env, and blocking doctor issues.
- Environment variables must normalize keys to uppercase before persistence and label hidden dirty values as overwrite drafts.
- Secret reveal must call the backend reveal endpoint; never show masked placeholders as real values.
- Logs and metrics must close EventSource/timers on destroy and on project id changes.
- Settings must not allow changes that backend rejects as immutable without explaining that constraint.

## Output Format

Produce findings in this order:

1. Blocking state bugs
2. High-risk UX state issues
3. Consistency repairs
4. Refactor opportunities
5. Suggested tests

For each finding, include file path, state owner, failure mode, user impact, and a concrete fix.
