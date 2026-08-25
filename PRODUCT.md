# Product

MyPaaS is a **single-host self-hosted deployment platform** for an owner developer or a small trusted team.

Its job is to make common deployment and operations work repeatable on a Linux server without hiding ownership of the host, container engine, data, or routing path.

## Current scope

MyPaaS can:

- deploy Git repositories with Dockerfile, Docker Compose, or static output;
- deploy public OCI images;
- inspect repository structure and configuration before creation;
- manage environment variables and project resource settings;
- manage routing through Caddy;
- expose deployment history, logs, metrics, and lifecycle actions;
- support rollback for compatible container-backed deployments;
- provide PostgreSQL provisioning, DB Studio Lite, backups, restore, and migration tooling;
- expose CLI, REST API, webhooks, and an optional local MCP bridge;
- use optional `mypaas-statd` telemetry with an engine-metrics fallback.

Fresh supported Linux installations default to rootful Podman through the Docker-compatible command/socket contract used by the control plane. Docker Engine remains an explicit compatibility mode.

## Boundaries

MyPaaS currently does **not** provide:

- multi-node scheduling or cluster orchestration;
- control-plane high availability;
- hostile multi-tenant isolation;
- automatic horizontal application scaling;
- private-registry credential management;
- supported in-place Docker-to-Podman state migration;
- a universal application-capacity guarantee.

Application and build capacity depend on the workload and on the CPU, memory, storage, network, database, and other processes sharing the host. Project count, concurrent users, RPS, or a particular VM size are not fixed capabilities of MyPaaS.

## Product principles

1. Keep deployment state and failure state explicit.
2. Prefer deterministic configuration over guessed automation.
3. Do not replace a healthy runtime with a failed deployment.
4. Keep recovery, rollback, backup, and cleanup operable.
5. Prefer simple single-host mechanisms over distributed-system complexity without a demonstrated need.
6. Keep public claims narrower than what the implementation and current evidence support.

## Security and operations

The MyPaaS API has privileged container-engine authority. The current trust model is therefore an owner or small trusted team, not mutually hostile tenants.

Host sizing, application architecture, provider availability, infrastructure security, and off-host recovery material remain operator responsibilities.

See [`docs/SECURITY_BOUNDARIES.md`](docs/SECURITY_BOUNDARIES.md) for the detailed trust model.
