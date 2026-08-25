# MyPaaS Real-World Compatibility Suite

This suite measures whether MyPaaS can correctly host representative real-world open-source workloads. It is a compatibility suite, not a throughput or capacity benchmark.

## Rules

- A `PASS` means the application deployed and its declared smoke checks succeeded on the tested MyPaaS host.
- A failure must be classified as a MyPaaS defect, an upstream application/configuration problem, a host-resource limit, or an intentional platform boundary.
- Do not convert a passing fixture into an RPS, concurrent-user, project-count, or hardware-capacity claim.
- Prefer upstream Docker/Compose deployment patterns and public OCI images. Compatibility manifests may only adapt host-specific details such as bind mounts, platform-owned routing, and secrets; they must not patch application code.
- Run heavy workloads separately on modest hosts. Resource exhaustion is not automatically a MyPaaS defect.

## Workload classes

| Class | Representative applications | What it exercises |
| --- | --- | --- |
| Simple image | Excalidraw | public OCI image, one HTTP runtime, routing |
| Source Dockerfile | drawDB | Git clone, Docker build, route activation |
| Stateful single service | Uptime Kuma, Meilisearch | named-volume persistence, runtime lifecycle |
| App + database | Umami, Ghost | Compose, SQL service, readiness, env |
| Realtime/stateful app | Directus | persistent storage plus WebSocket-capable routing |
| Developer platform | Forgejo | persistent application with an additional-port boundary |
| Multi-service application | NocoDB | app, worker, PostgreSQL, Redis, named volumes |
| Automation platform | n8n | state, background execution, environment configuration |
| Document platform | Paperless-ngx | web app, PostgreSQL, Redis, persistent media/data |
| Knowledge platform | Outline | app, PostgreSQL, Redis, required secrets/auth configuration |
| Agent gateway | OpenClaw | OCI image, persistent state, env-heavy setup, security boundaries |
| Heavy media platform | Immich | multiple runtimes, database, cache, ML process, storage |
| Heavy all-in-one platform | Appsmith | large application footprint and persistent state |
| Multi-port storage | MinIO | capability boundary for applications needing more than one public port |

The machine-readable source of truth is [`catalog.json`](catalog.json).

## Result vocabulary

- `untested` — catalogued but not yet run on the target MyPaaS host.
- `pass` — deployment and declared checks completed successfully.
- `fail-platform` — reproducible MyPaaS defect.
- `fail-app` — upstream application/configuration problem unrelated to MyPaaS.
- `fail-resource` — host CPU/RAM/disk capacity was the limiting factor.
- `blocked` — the application requires a capability MyPaaS intentionally does not provide.

Live results belong in run artifacts, issues, or pull requests. They should not be committed as permanent capacity claims.
