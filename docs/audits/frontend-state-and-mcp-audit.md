# MyPaas Frontend State And MCP Audit

Date: 2026-08-10

## Scope

This audit used the new project-local skills:

- `.agents/skills/mypaas-dashboard-state-audit`
- `.agents/skills/mypaas-mcp-goat`

Reviewed current source files:

- `frontend/src/routes/projects/new/+page.svelte`
- `frontend/src/routes/projects/[id]/env/+page.svelte`
- `frontend/src/routes/projects/[id]/settings/+page.svelte`
- `frontend/src/lib/api/index.ts`
- `backend/cmd/mcp/main.go`
- `.agents/mcp/mypaas/mcp_config.json`

## Cleanup

No exact `impacable`, `impeccable`, or `desing` source references were found. Local generated backend binaries were present as ignored build artifacts and should stay out of repository work:

- `backend/api.exe`
- `backend/cli.exe`
- `backend/mcp.exe`
- `backend/mcp.exe~`
- `backend/tmp_build`

## Frontend State Findings

### Blocking State Bugs

None found in the reviewed source slice.

### High-Risk UX State Issues

1. Settings allows project rename even though backend validation rejects renames.

File: `frontend/src/routes/projects/[id]/settings/+page.svelte`

State owner: local `name` draft plus server `project.name`.

Failure mode: the UI shows a normal editable project name and subdomain preview, but the backend now keeps project names immutable through `UpdateValidated`. A user can spend effort editing a value that cannot be saved.

Fix: render project name as read-only identity, or show an explicit "project name is immutable after creation" note. Remove `name !== project.name` from dirty-state calculation unless a migration later supports runtime renames.

2. Settings base directory is free text only and lacks repository validation feedback.

File: `frontend/src/routes/projects/[id]/settings/+page.svelte`

State owner: `baseDirectory` local draft.

Failure mode: project creation validates repo URL, branch, and base directory before persistence, but settings can submit a new base directory without previewing whether that path exists or whether runtime files still resolve.

Fix: add an `inspectRepository` pass for Git projects in settings, keyed by `repoUrl + branch + baseDirectory`, and block save when validation is stale or failed. A directory picker from repo tree would be useful, but should be added without removing registry project handling.

3. Project creation has too much page-local mutable state.

File: `frontend/src/routes/projects/new/+page.svelte`

State owner: one large `form` object plus many peer state variables (`composePlan`, `repoTree`, `envDrafts`, `staticFrontendCandidates`, `appPortSource`, etc.).

Failure mode: state reset rules are spread across `chooseSourceType`, `chooseDeployMode`, `clearDetectedSourceState`, `resetRepositoryInspection`, `handleBranchChange`, and detection handlers. This makes regressions likely when adding source types or runtime options.

Fix: extract a route-local reducer-style module such as `frontend/src/lib/project-create/state.ts` with explicit events: `sourceChanged`, `repoChanged`, `branchChanged`, `baseDirectoryChanged`, `inspectionSucceeded`, `detectionSucceeded`, `modeChanged`, and `submitStarted/Finished`. Keep Svelte page markup as orchestration, not the state machine.

### Consistency Repairs

1. API client throws exceptions even though the project standard says API calls should return `{ data, error }`.

File: `frontend/src/lib/api/index.ts`

Impact: pages must use `try/catch` for expected domain errors. That works today, but it diverges from the stated frontend standard and spreads error-display decisions across pages.

Fix: either update the standard to allow thrown `ApiError`, or wrap API methods with a typed result helper and migrate high-churn pages first.

2. Environment import state is strong, but add and edit share different pending names.

File: `frontend/src/routes/projects/[id]/env/+page.svelte`

Impact: `adding`, `savingNewVar`, `savingChanges`, `savingImport`, and `importing` are individually clear, but the page would benefit from a single environment draft model for tests and reset behavior.

Fix: extract import parsing and row mutation into a small module and add unit tests for dirty/reveal/import interactions.

3. Settings stores service resources as JSON string.

File: `frontend/src/routes/projects/[id]/settings/+page.svelte`

Impact: parsing errors are handled, but JSON-as-string makes dirty checks noisy and easy to desynchronize after save.

Fix: keep string editing in the UI, but add a derived `serviceResourcesParseResult` and reset the string from the server after successful save.

## Suggested Frontend Tests

- Settings: project name field cannot be edited or does not mark settings dirty.
- Settings: changing base directory requires current repository validation before save.
- New project: registry source can submit without repository validation.
- New project: Git source cannot submit when repo inspection key is stale.
- Env vars: dirty hidden value cannot reveal until saved or discarded.
- Env vars: import overwrite requires explicit confirmation and is blocked while existing dirty drafts exist.

## MCP Inventory

Current MCP server: `backend/cmd/mcp/main.go`.

Current tools:

- `list_projects`
- `deploy_project`
- `get_project`
- `stop_project`
- `start_project`
- `create_project`

## MCP Findings

### High-Risk Gaps

1. `create_project` schema is stale.

Current tool lacks current project creation capabilities:

- `sourceType`
- `imageRef`
- `baseDirectory`
- `staticFrontendPath`
- `composeFilePath`
- `composeOverridePaths`
- `composeProfiles`
- `composeWorkdir`
- `envVars`
- resource service overrides

It also defaults `sharedPostgres` to `true`, while the tool description says agents should enable it only when PostgreSQL is explicitly required. This can create unexpected database provisioning.

2. MCP cannot inspect before creating.

Missing tools:

- `inspect_repository`
- `detect_compose`

Without those, agents are asked to infer deploy configuration manually while the product already has backend detection endpoints.

3. MCP cannot observe deployment outcomes.

Missing tools:

- `list_deployments`
- `get_deployment`
- `get_logs`
- `get_metrics_snapshot`

The MCP can trigger deployment but cannot verify whether it succeeded from inside the same interface.

4. MCP lacks secret-safe environment workflows.

Missing tools:

- `list_env_vars`
- `set_env_vars`
- `delete_env_var`
- `reveal_env_var` with explicit confirmation

This blocks practical agent-driven deployments that require env configuration.

### Implementation Risks

- `http.Client{}` has no timeout.
- Handlers read `map[string]interface{}` directly instead of typed argument structs.
- Error results flatten backend error JSON into text and lose structured `code`.
- Destructive tools do not have confirmation phrases because delete/reset/regenerate tools do not exist yet.
- Tool descriptions contain long behavioral instructions instead of encoding guidance in schemas and separate inspect tools.

## MCP Improvement Roadmap

1. Repair current tools.

- Add HTTP timeout.
- Return structured JSON with `ok`, `data`, and `error`.
- Change `create_project` default `sharedPostgres` to false unless explicitly supplied.
- Add current fields to `create_project`.

2. Add discovery tools.

- `inspect_repository(repoUrl, branch?, baseDirectory?)`
- `detect_compose(repoUrl, branch, baseDirectory?)`
- `get_quota()`
- `get_host_stats()`

3. Add operational read tools.

- `list_deployments(project_id, limit?, offset?)`
- `get_deployment(deployment_id)`
- `get_logs(project_id, tail?)`
- `get_metrics_snapshot(project_id)`

4. Add configuration tools.

- `update_project_settings`
- `list_env_vars`
- `set_env_vars`
- `delete_env_var`

5. Add guarded high-risk tools.

- `restart_project`
- `rollback_deployment`
- `delete_project`
- `reveal_env_var`
- `regenerate_mcp_token`

Each guarded tool should require an explicit confirmation field that includes the project name, env key, or token action.

## Recommended State Management Direction

Keep Svelte stores for cross-route UI concerns only: theme, sidebar, session/toast, and possibly live deployment stream summaries. Keep route-specific drafts local, but move complex draft transitions into pure TypeScript modules with tests.

Near-term structure:

- `frontend/src/lib/project-create/state.ts`
- `frontend/src/lib/project-settings/state.ts`
- `frontend/src/lib/env/state.ts`

Each module should expose typed state, typed events, derived selectors, and reset helpers. The Svelte pages should bind controls, call API methods, and render state, but not own every transition rule inline.
