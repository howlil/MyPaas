# Platform settings semantics validation

This change intentionally keeps Admin Settings limited to controls with an authoritative live runtime consumer.

Validation target before merge:

- `go test ./...`
- frontend unit tests
- Svelte/TypeScript checks
- production frontend build

The Create Project Playwright audit is intentionally not modified by this branch. Project resource defaults continue to come from the existing runtime profile system.
