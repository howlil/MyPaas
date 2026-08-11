# MyPaaS Security Boundaries

MyPaaS is a single-host deployment platform. Its security model separates the control plane from project workloads, but it does not treat the control-plane API itself as a sandboxed or least-privilege container-engine client.

## Container-engine socket

The API mounts the configured Docker-compatible engine socket because deployment orchestration, runtime lifecycle, inspection, logs, metrics fallback, image management, Compose operations, and runtime route resolution require engine authority.

Access to that socket is effectively host-level container-engine authority. Compromise of the API process must therefore be treated as compromise of the MyPaaS host boundary, even though the API container drops Linux capabilities and enables `no-new-privileges`.

The production API container therefore:

- is never exposed directly on a public host interface;
- joins only the control-plane network;
- uses `no-new-privileges:true`;
- drops all ambient container capabilities;
- does not pass the engine socket to project workloads.

A socket proxy is intentionally not introduced yet. It would add another privileged component and a large engine-API authorization surface. Revisit that choice only if the deployment engine can be reduced to a small, auditable API subset.

## Network separation

Production uses three external container networks:

- `CONTROL_NETWORK` (default `mypaas-control`) for API, dashboard, Cloudflare Tunnel, Caddy control-plane connectivity, and PostgreSQL control-plane access;
- `PROJECT_NETWORK` (default `mypaas-projects`) for ordinary MyPaaS-managed workloads and shared PostgreSQL access;
- `ROUTING_NETWORK` (default `mypaas-routing`) for Caddy's application data plane and only those runtime containers that MyPaaS explicitly attaches while activating a public route.

The API, dashboard, and Cloudflare Tunnel join only the control network. PostgreSQL intentionally joins control + project for the shared PostgreSQL feature. Caddy intentionally joins control + routing, but not the project network.

This means an ordinary project workload does not automatically receive container-network reachability to API, dashboard, Cloudflare Tunnel, or Caddy. A routed runtime receives a second attachment to `ROUTING_NETWORK` only when MyPaaS activates its public route.

The three network names must be distinct. Production verification enforces those memberships to catch topology drift.

## Runtime route resolution

Production uses `CADDY_UPSTREAM_HOST=runtime`. MyPaaS keeps each allocated host port as a stable runtime identity key, but Caddy does not send application traffic through that published host port.

When a dynamic route is created or reconciled, the API:

1. lists running containers through the Docker-compatible engine;
2. inspects them in one batched engine call;
3. finds the container whose published binding owns the project's allocated host port;
4. confirms that container is attached to `PROJECT_NETWORK` and derives the corresponding internal container port;
5. attaches the selected container to `ROUTING_NETWORK` with the explicit alias `mypaas-port-<allocated-port>` when necessary;
6. writes `mypaas-port-<allocated-port>:<internal-port>` into the Caddy reverse-proxy route.

Explicit network aliases are used instead of compatibility-layer container IP fields or short-container-ID DNS behavior. This avoids Docker/Podman host-port hairpin behavior and keeps the routing identity independent of container rename. That matters for rolling Dockerfile/image deployments, where the replacement container is routed before its temporary name is renamed to the stable project name.

If a container is already attached to `ROUTING_NETWORK` without the expected managed alias, MyPaaS refreshes only that secondary routing attachment before writing the route. Route resolution is fail-closed; MyPaaS does not silently fall back to an arbitrary host address.

The published host binding remains for deterministic runtime identification and existing lifecycle/accounting semantics. It is not the Caddy data path.

## Caddy administration

The production Caddyfile configures:

```text
admin unix//run/mypaas/caddy-admin.sock
```

There is no production mapping for TCP port `2019`. The Go Caddy client supports the Unix admin endpoint directly through an HTTP transport that dials the Unix socket. The API and Caddy share `/run/mypaas`; project workloads and routed workloads do not receive that host mount.

This separates Caddy's application data plane from its privileged configuration plane. A routed workload may share `ROUTING_NETWORK` with Caddy's normal HTTP listener, but it does not receive the Caddy Admin socket.

## User Compose execution

Repository Compose files are not passed directly to the engine as trusted configuration. MyPaaS renders the final Compose model and enforces its host-isolation policy before execution.

The security policy rejects host-escape features including privileged containers, host/container namespace sharing, host bind mounts, engine socket mounts, devices, added capabilities, GPUs, custom runtimes, external networks/volumes, unsafe build entitlements, build SSH/secrets, and privileged lifecycle hooks. MyPaaS also strips repository-defined host ports and container names before applying its managed runtime override.

This policy is an important single-host isolation boundary, but it is not equivalent to a VM, microVM, or Kubernetes multi-tenant sandbox.

## Native telemetry daemon

`mypaas-statd` is host-native by design. It reads cgroup v2 data and exposes a bounded protocol over `/run/mypaas/statd.sock`. The API receives the statd socket directory but does not receive host `/proc` or `/sys/fs/cgroup` mounts.

Statd failure is non-fatal: MyPaaS falls back to the Docker-compatible metrics path. Production metrics expose statd availability and fallback/error counters so that fallback is observable instead of silent.

## Trust model

The current production target is a single administrative owner or a small trusted team deploying workloads onto one host. Before treating arbitrary external users as mutually untrusted tenants, additional isolation work would be required, potentially including per-tenant VM/microVM boundaries, stronger database isolation, host resource governance, and a narrower control-plane engine authority model.
