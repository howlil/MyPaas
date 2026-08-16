# Product

## Release status

MyPaas is currently a **beta single-host self-hosted PaaS**.

The beta is intended for an owner developer or a small trusted team that wants a repeatable deployment control plane on a Linux VM without giving up ownership of the host, container engine, persistent data, and routing path.

The release has completed the mandatory beta-readiness gates recorded in `docs/engineering/beta-readiness-gates.md`. That qualification is evidence for the tested release lineage, not a promise of hyperscaler-grade availability or arbitrary workload capacity.

## Target users

MyPaas is designed for developers and small technical teams who routinely:

- deploy Git repositories or public OCI images;
- inspect build/deployment failures;
- manage environment variables;
- watch logs and runtime metrics;
- roll back a bad container-backed deployment;
- operate Compose applications with persistent data;
- inspect supported project databases through DB Studio;
- maintain a constrained self-hosted VM.

## Product purpose

MyPaas makes self-hosted deployment feel closer to a managed PaaS while preserving infrastructure ownership.

It connects Git repositories to Dockerfile, Docker Compose, or static deployments; supports public registry images; manages routing through Caddy and Cloudflare Tunnel; tracks deployment history; exposes logs and metrics; and keeps common lifecycle, backup, recovery, and database-inspection actions in one control plane.

Success means a developer can deploy, diagnose, recover, and maintain a project without repeatedly rebuilding the same infrastructure glue.

## Current product boundaries

The product deliberately does **not** claim to be:

- a Kubernetes replacement;
- a multi-node cluster scheduler;
- a highly available control plane;
- a hostile multi-tenant isolation boundary;
- a managed public cloud service;
- an unlimited-capacity deployment platform.

Current boundaries include:

- one Linux host per MyPaas installation;
- owner / small trusted-team access model;
- Docker-compatible engine contract with qualified Docker Engine and Podman operation;
- public OCI registry deployment only; private registry credentials are outside the current implementation;
- no supported in-place Docker-to-Podman state migration;
- performance and project capacity depend on host resources and workload shape.

## Product principles

1. **Deployment state first.** Current state, next action, and risk should be obvious before the user acts.
2. **Honest automation.** Detected values must be distinguishable from fallbacks and manual configuration.
3. **Fail closed.** Stale analysis, incomplete configuration, unhealthy replacements, and invalid recovery paths must not be presented as success.
4. **Recovery is a first-class flow.** Retry, rollback, reconnect, restore, revoke, and cleanup states should be visible and trustworthy.
5. **Single-host operational clarity.** Prefer simple, mature mechanisms that fit the current architecture over distributed-system complexity the product does not need.
6. **Evidence before claims.** Performance, compatibility, and release-readiness claims should point back to tests or recorded qualification evidence.

## Brand personality

Quiet, capable, and operationally precise.

The interface should feel modern and controlled rather than decorative: dense enough for repeated operational use, legible under pressure, and explicit about state. Avoid generic SaaS ornamentation that competes with deployment information.

## Anti-references

Avoid:

- oversized marketing-style composition inside the dashboard;
- decorative gradients and glassmorphism used as primary structure;
- nested card stacks without information hierarchy;
- vague spinners that hide actionable state;
- wizard steps that unnecessarily slow source configuration;
- interfaces that imply ports, secrets, runtime type, or readiness are known when they are only inferred or stale.

## Accessibility

Target WCAG AA contrast for text and controls. Preserve keyboard access, visible focus, semantic controls, reduced-motion-safe interactions, and status copy that does not rely on color alone.

Operational screens should remain usable on laptop and mobile viewports without clipping primary actions or hiding required configuration.
