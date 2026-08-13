# ADR-021: Project metrics separate runtime telemetry from storage capacity

## Status

Accepted

## Context

MyPaas exposes fast runtime metrics for project containers and host telemetry through `mypaas-statd`. The dashboard previously presented CPU, memory, uptime, and host storage with similar visual weight even though they describe different concepts.

The current statd runtime snapshot contains CPU, memory, and PID/cgroup state. It does not contain project-scoped filesystem usage. Host storage telemetry describes the VM root filesystem and therefore must not be presented as storage consumed by an individual project.

Compose projects may also use engine-managed named/external volumes that are deliberately outside the built-in migration contract. Treating image layers, build cache, engine-local volumes, and host root usage as one "project storage" number would be misleading.

## Decision

Project observability uses these semantics:

- CPU and memory are fast runtime resource telemetry. MyPaas prefers `mypaas-statd` when configured and falls back to the container-engine metrics path when statd is unavailable.
- Uptime and service name are runtime context, not resource-utilization metrics.
- Host storage is a capacity metric: used bytes versus total/available bytes. It is rendered as a capacity bar rather than a fast-changing sparkline.
- Project storage means MyPaas-managed persistent data attributable to one project. Host root-disk usage is never substituted for this value.
- Until MyPaas has an authoritative project-scoped persistent-data collector, project UI reports storage as not measured instead of fabricating a number.
- A future project-storage collector should run on a slower cadence than CPU/memory and must explicitly define what it includes. It may be implemented through a stable project-managed path contract or a future statd protocol extension.

The project Overview answers whether the normal runtime is healthy. The dedicated Metrics page remains a diagnostic surface for service-level runtime data, Compose service selection, and optional Cloudflare traffic analytics rather than duplicating the Overview without additional context.

## Consequences

- Storage visualization matches its capacity semantics.
- Users can see that runtime metrics prefer statd without being told that every individual sample definitely came from statd when fallback is possible.
- Project storage remains truthful even though the current collector cannot provide a project-scoped byte count.
- Uptime is moved into runtime context instead of being presented as a third resource meter.
- The Metrics route remains justified as a diagnostic/deep-inspection page, especially for Compose and Cloudflare-enabled projects.
