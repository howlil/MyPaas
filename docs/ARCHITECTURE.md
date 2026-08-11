# Architecture — MyPaaS

> Current single-host architecture, trust boundaries, deployment flow, and runtime observability.

**Status:** production-hardening baseline  
**Last updated:** 2026-08-11

---

## 1. High-Level Architecture

```text
Internet
   |
Cloudflare Tunnel
   |
 Caddy
   |----------------------> SvelteKit dashboard
   |----------------------> Go API
   |                          |
   |                          +--> PostgreSQL
   |                          +--> Caddy Admin Unix socket
   |                          +--> Docker-compatible CLI/socket
   |                                   |
   |                                   +--> Docker Engine
   |                                   `--> Podman socket compatibility
   |
   +----------------------> static project files
   |
   `----------------------> routed container runtime

Host
 `--> mypaas-statd (systemd, cgroup v2, Unix socket)
```

MyPaaS targets one Linux VM. It is not a Kubernetes scheduler and does not implement a second Podman-specific orchestration backend. The Go control plane uses the Docker-compatible command/socket contract; Podman production hosts expose that contract through `podman.socket`.

`mypaas-statd` is deliberately host-native. It is a small cgroup v2 telemetry daemon consumed over `/run/mypaas/statd.sock`; API/business orchestration remains in Go.

---

## 2. Production Components

### 2.1 Go API

Responsibilities include:

- GitHub OAuth, authorization, and project state;
- repository inspection and deployment orchestration;
- Dockerfile, Compose, registry-image, static, and hybrid deployment flows;
- lifecycle, rollback, backups, cleanup, DB Studio, CLI/MCP APIs;
- Caddy route reconciliation;
- log/SSE delivery;
- statd metrics integration with Docker-compatible fallback.

The API mounts the Docker-compatible engine socket. That socket is a host-authority trust boundary; the API container therefore drops Linux capabilities and enables `no-new-privileges`, but the socket itself still means an API compromise must be treated as host compromise.

### 2.2 Caddy

Caddy handles:

- dashboard/API ingress from Cloudflare Tunnel;
- static project serving;
- dynamic project reverse-proxy routes.

Production Caddy administration is **Unix-socket only**:

```text
/run/mypaas/caddy-admin.sock
```

TCP admin port `2019` is not part of the production contract.

### 2.3 PostgreSQL

PostgreSQL stores MyPaaS control-plane state and can optionally provision project-specific shared databases/users. Its control-plane data uses the production Compose named volume; project bind-mount state remains under `/var/lib/mypaas`.

### 2.4 Container Engine

The runtime abstraction intentionally remains CLI-oriented:

```text
Go control plane
    |
 docker CLI / docker compose
    |
 Docker-compatible socket
    |
 Docker Engine or rootful Podman
```

Compose is rendered, security-validated, sanitized, and then combined with a MyPaaS-managed override before execution.

### 2.5 mypaas-statd

`mypaas-statd` reads cgroup v2 CPU/memory/PID counters on the host and serves bounded cached snapshots over a local Unix socket. It removes repeated Docker/Podman process spawning from the normal metrics hot path while preserving the existing Docker-compatible fallback.

See `docs/STATD.md` for the protocol, benchmark evidence, release model, and operational metrics.

---

## 3. Network Model

Production uses three distinct external networks.

| Network | Members | Purpose |
| --- | --- | --- |
| `CONTROL_NETWORK` | API, dashboard, cloudflared, PostgreSQL, Caddy | control-plane communication |
| `PROJECT_NETWORK` | user workloads, PostgreSQL | workload/service communication and optional shared DB access |
| `ROUTING_NETWORK` | Caddy + explicitly routed runtimes | public application data plane |

The normal project workload is not attached to the control network. Caddy is not attached to the general project network. A runtime gets a routing-network attachment only when MyPaaS activates its public route.

PostgreSQL is intentionally dual-homed on control + project because shared PostgreSQL is an explicit platform feature.

For the detailed security rationale, see `docs/SECURITY_BOUNDARIES.md`.

---

## 4. Dynamic Runtime Routing

Allocated host ports remain part of the deployment/lifecycle identity model, but production Caddy does not hairpin through those published ports.

When a route is activated, the API:

1. lists running containers through the Docker-compatible engine;
2. batch-inspects the candidates;
3. finds the container whose published binding owns the project's allocated host port;
4. verifies that runtime belongs to `PROJECT_NETWORK` and determines its internal application port;
5. attaches it to `ROUTING_NETWORK` with the deterministic alias `mypaas-port-<allocated-port>`;
6. configures Caddy to proxy to `mypaas-port-<allocated-port>:<internal-port>`.

This avoids relying on Docker/Podman bridge-gateway hairpin behavior, compatibility-layer IP fields, or container-name stability. The alias is tied to the allocated deployment port, so a Dockerfile/image replacement can be routed before its temporary container name is renamed.

Route resolution is fail-closed. Failure to identify or attach the intended runtime does not silently fall back to an arbitrary host address.

Static routes do not use this runtime path; Caddy serves their files from the host-managed static directory.

---

## 5. Deployment Flow

```text
1. Trigger
   GitHub webhook / manual deploy / rollback / registry deploy

2. Persist + queue
   create deployment state
   bounded in-memory worker concurrency

3. Prepare source
   clone/checkout or pull registry image
   resolve base directory / Compose layout / env files

4. Build where required
   docker build for Dockerfile projects
   docker compose up --build for Compose
   bounded ephemeral Node builder for static SPA builds

5. Start runtime
   Dockerfile/image: replacement container lifecycle
   Compose: sanitized config + managed override
   Static: atomic static release switch

6. Route
   runtime project -> managed routing alias + Caddy route
   static project  -> Caddy file-server route

7. Commit state
   active deployment + running state

8. Cleanup
   temporary workspace / old runtime/image state as applicable
```

The static Node builder currently has a simple safety ceiling of 2 GiB RAM, 2 CPUs, and 512 PIDs. The deployment context still supplies the outer build timeout.

---

## 6. Compose Security Boundary

Repository Compose is untrusted input. MyPaaS evaluates the rendered configuration before execution and rejects host-escape features including:

- privileged containers;
- host/container namespace sharing;
- Docker/Podman socket mounts;
- host bind mounts;
- devices and GPUs;
- added Linux capabilities;
- custom runtimes;
- external networks/volumes;
- unsafe build entitlements, SSH, and build secrets;
- privileged lifecycle hooks.

Repository-defined host ports and container names are stripped before the managed runtime override is applied.

Compose subprocesses receive a fail-closed host-environment allowlist. Project variables are provided through the project's generated `--env-file`; arbitrary control-plane credentials are not inherited simply because they exist in the API process environment.

This is a strong single-host policy boundary, not a substitute for VM/microVM isolation between mutually hostile tenants.

---

## 7. Logs and Metrics

### 7.1 Logs

Current project log collection still uses the Docker-compatible CLI path. Compose collection discovers services and reads recent logs per service, and live project events are delivered through SSE.

`benchmarks/log_path.py` mirrors the current Compose log-command path and records request latency, subprocess count, and child CPU. It exists specifically to decide whether a future persistent/native log collector is justified. No `mypaas-logd` architecture should be introduced without measured evidence.

### 7.2 Runtime metrics

Preferred live runtime path:

```text
API
 |
Unix socket
 |
mypaas-statd
 |
cgroup v2
```

The handler keeps bounded runtime metadata so steady-state statd reads do not repeatedly rediscover runtime PIDs through Docker/Podman commands. Statd failure is non-fatal and falls back to the Docker-compatible metrics implementation.

The API exposes low-cardinality statd availability/fallback counters through `/metrics`, and transition-aware logging prevents repeated SSE fallback from flooding production logs.

---

## 8. Persistence and Recovery

Host-managed persistent paths include:

```text
/var/lib/mypaas/volumes
/var/lib/mypaas/compose
/var/lib/mypaas/static
/var/lib/mypaas/backups
```

VM export performs storage preflight before downtime, quiesces running runtimes without changing desired database state, creates the package, restores runtimes, and only then marks the package ready.

Engine-managed Compose named/external volumes are rejected by migration preflight because they cannot be assumed portable between Docker Engine and Podman. See ADR-019.

Migration import provisions the control/project/routing external networks before bringing up the production Compose stack.

---

## 9. Runtime and Operational Verification

Repository CI covers:

- Go tests;
- Go race detection;
- frontend unit/type/build checks;
- Bash syntax;
- script regression tests;
- benchmark-harness unit tests;
- production Compose rendering;
- Docker/Podman command-contract smoke testing.

`scripts/verify-production.sh` additionally checks the live production topology, API readiness, statd service/socket when configured, the Caddy Admin Unix socket, and the absence of a published Caddy TCP admin endpoint.

Real-host performance claims and final production-like Podman behavior remain staging/benchmark concerns rather than being inferred solely from CI.

---

## 10. Scope Boundaries

Current architecture intentionally remains:

- single-host;
- owner/small trusted-team oriented;
- no Kubernetes;
- no multi-node scheduler or HA control plane;
- no private-registry credential manager;
- Docker-compatible Podman support rather than a separate native Podman backend;
- no native helper daemon unless profiling demonstrates a concrete hot-path benefit.

For mutually untrusted external tenants, stronger VM/microVM-style isolation and a narrower engine-authority boundary would be required.

---

## References

- `docs/STATD.md` — native telemetry integration and benchmark evidence
- `docs/SECURITY_BOUNDARIES.md` — control-plane/runtime trust boundaries
- `docs/adr/ADR-019-migration-safety-boundaries.md` — migration fail-closed decisions
- `docs/PRD.md` — product requirements
- `PRODUCT.md` — product direction
- `AGENTS.md` — engineering constraints
