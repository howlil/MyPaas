---
name: mypaas-mcp-goat
description: Design, create, and audit high-value MCP servers and tools for MyPaas. Use when reviewing MyPaas MCP capability, adding MCP tools, improving MCP schemas, checking tool safety, or planning helpful agent-facing automation for projects, deployments, logs, metrics, env vars, admin settings, backups, migrations, database studio, and audits.
---

# MyPaas MCP GOAT

Use this skill to make MyPaas MCP tools useful, safe, and agent-friendly. Treat MCP as an operator interface over MyPaas, not a thin random wrapper around HTTP endpoints.

## Capability Design

Each MCP tool must have:

- One clear user intent.
- A strict input schema with explicit required fields.
- Defaults that match MyPaas product rules.
- Output that summarizes the result and includes machine-readable JSON when useful.
- Safe behavior for destructive or expensive actions.
- Error messages that preserve backend `code` and `message`.

## MyPaas Tool Coverage Targets

Baseline project operations:

- `list_projects`
- `get_project`
- `create_project`
- `deploy_project`
- `start_project`
- `stop_project`
- `restart_project`
- `delete_project` with confirmation phrase

Operational inspection:

- `list_deployments`
- `get_deployment`
- `rollback_deployment` with explicit target id
- `get_logs` with tail limit
- `get_metrics_snapshot`
- `get_quota`
- `get_host_stats`
- `list_audit_logs`

Configuration:

- `inspect_repository`
- `detect_compose`
- `update_project_settings`
- `list_env_vars`
- `set_env_vars`
- `delete_env_var`
- `reveal_env_var` only when explicitly requested
- `get_admin_settings`
- `regenerate_mcp_token` only with confirmation

Database and maintenance:

- `get_db_status`
- `list_db_schemas`
- `list_db_tables`
- `list_db_rows` with safe limits
- `prepare_migration`
- `get_migration_status`

## Audit Checklist

1. Compare MCP tools against frontend API methods in `frontend/src/lib/api/index.ts` and backend routes.
2. Check whether tools support current product capabilities: registry image source, base directory, static frontend path, Compose config, env vars, quota, logs, metrics, and admin settings.
3. Check schema fidelity:
   - No stale defaults like shared Postgres enabled by default unless the UI/backend does the same.
   - Include `sourceType`, `imageRef`, `baseDirectory`, `staticFrontendPath`, Compose fields, and resource fields for project creation.
   - Use `project_id` consistently for MCP inputs, but map correctly to backend URLs.
4. Check safety:
   - Destructive actions require confirmation tokens.
   - Secret reveal and token regeneration require explicit request wording.
   - Logs and row queries enforce limits.
5. Check implementation quality:
   - Reuse request helpers.
   - Set timeouts on HTTP clients.
   - Return structured errors.
   - Avoid `map[string]interface{}` drift when typed request structs are practical.
   - Add table-driven handler tests for payload construction and error handling.

## Output Format

Produce:

- Current MCP inventory
- Missing high-value tools
- Stale or unsafe schema details
- Implementation risks
- Prioritized MCP roadmap
- Minimal acceptance tests for each proposed tool group
