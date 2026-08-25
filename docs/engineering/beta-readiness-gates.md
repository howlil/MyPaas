# Beta Runtime Verification

This document records controlled checks used while hardening the MyPaaS beta. It is a reliability record for platform behavior, not a capacity benchmark.

## What these checks mean

A passing check means the tested MyPaaS behavior worked in the named controlled scenario. It does **not** establish a universal project count, concurrent-user count, RPS ceiling, application size, or minimum production server specification.

Application capacity depends on the application itself and on available CPU, memory, storage, network, database behavior, build requirements, and other workloads sharing the host.

## Retained runtime checks

| Area | What was verified |
| --- | --- |
| Update / release safety | Controlled update, missing-image safety, rollback, health verification, build identity, and route reconciliation. |
| Backup / restore | Fresh-host restoration of control-plane data, configuration, static artifacts, managed persistent data, routes, encrypted environment usability, deployment history, and DB Studio state. |
| Concurrent deployment reliability | Concurrent create/deploy/redeploy, intentional failure, protected routes, webhook activity, and final runtime/port consistency. |
| Image and cache retention | Cleanup scope, protection of running/rollback images and persistent volumes, dry-run/apply behavior, and post-cleanup deployment/rollback. |
| Create Project contract | Static, Dockerfile, Compose, subdirectory, public registry/GHCR, required environment, missing-port, stale-analysis, timeout, and invalid-repository behavior. |
| DB Studio | PostgreSQL, MySQL, and MariaDB Compose connectivity, project-network resolution, read-only default, and expiring write sessions. |

These checks remain useful regression targets when their corresponding runtime paths change.

## Historical defects found during qualification

The qualification work found real product defects that were subsequently fixed, including:

- Compose empty-health handling causing deployment timeout and port-state divergence;
- Dockerfile side-by-side redeploy attempting to reuse an active runtime port;
- a DB Studio test fixture incorrectly treating an immutable commit SHA as a branch.

These are retained as regression history. They should not be converted into broader performance or scalability claims.

## Evidence handling

Generated VM logs, screenshots, load output, host snapshots, and other run artifacts are test evidence, not product documentation. They should normally stay outside the source tree or be attached to the specific issue, pull request, or release investigation that needs them.

For a controlled runtime check, record only what is needed to reproduce the result:

- tested Git SHA;
- relevant environment shape;
- scenario and expected behavior;
- pass/fail result;
- failure details when applicable.

Never store credentials, decrypted environment values, cookies, or other secrets in evidence.

## Regression rule

Re-run a controlled runtime check when a change materially touches the behavior it covers. Do not re-run unrelated scenarios merely to preserve a historical gate count.

A failure caused by application resource demand or insufficient host capacity must be classified separately from a MyPaaS correctness failure unless evidence shows the platform itself caused the fault.

## Current boundary

MyPaaS remains a single-host self-hosted platform for an owner developer or a small trusted team. It is not a multi-node HA scheduler, a hostile multi-tenant isolation boundary, or a guarantee that arbitrary workloads will fit on a given server.
