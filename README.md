# MyPaas — Self-Hosted Deployment Platform

> Deploy Git repositories and pre-built container images to your own VM with managed routing, logs, deployments, and rollback.

MyPaas is a lightweight self-hosted PaaS for developers and small teams that want a simple Vercel/Railway-style deployment workflow without giving up control of the underlying server.

It runs on Docker, uses Caddy for project routing, integrates with Cloudflare Tunnel for public access, and provides a SvelteKit dashboard backed by a Go API and PostgreSQL.

## Features

- **Git repository deployments** — deploy Dockerfile, Docker Compose, and static applications.
- **Monorepo/subdirectory support** — deploy from a repository root or a selected base directory such as `apps/api` or `docs`.
- **Container Registry deployments** — pull and run public OCI images from Docker Hub, GHCR, or another compatible public registry.
- **Automatic runtime detection** — inspect repository structure and infer supported deployment modes.
- **Git push deployments** — trigger deployments through project webhooks.
- **Automatic routing** — each project receives a subdomain and is routed through Caddy.
- **Cloudflare integration** — expose the dashboard and project wildcard domain without opening inbound application ports.
- **Deployment history and rollback** — inspect previous deployments and redeploy a previous Git revision or image.
- **Realtime operations** — project logs, runtime metrics, start/stop/restart, environment variables, and deployment status.
- **Resource controls** — configure project memory and CPU limits with quota enforcement.
- **GitHub OAuth** — authenticate approved users through GitHub.
- **Production VM installer** — bootstrap a fresh Ubuntu/Debian host from one command.
- **Safe self-updates** — optional systemd-based updates that keep Git source and GHCR images on the same commit revision.

> Container Registry deployment currently targets **public images**. Private registry credential management is intentionally outside the current MVP.

---

## Quick Start

### Production VM

For a fresh Ubuntu/Debian VM:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

The bootstrap process:

1. installs Git when needed;
2. checks out `main` into `~/MyPaas`;
3. starts the browser setup wizard;
4. checks Docker + Docker Compose and installs Docker when supported;
5. writes the production `.env`;
6. prepares persistent host directories under `/var/lib/mypaas`;
7. runs database migrations;
8. starts `docker-compose.prod.yml`;
9. verifies the production stack.

During interactive setup, the wizard remains bound to `127.0.0.1`. The installer can expose it temporarily through a token-protected Cloudflare Quick Tunnel and removes that tunnel after configuration is saved.

If Quick Tunnel startup is unavailable, use SSH forwarding:

```bash
ssh -L 8787:127.0.0.1:8787 <user>@<vm-ip>
```

### Non-interactive production install

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env \
  INSTALL_WIZARD=false \
  PUBLIC_DOMAIN=mypaas.example.com \
  OWNER_EMAIL=you@example.com \
  GITHUB_CLIENT_ID=your_client_id \
  GITHUB_CLIENT_SECRET=your_client_secret \
  CLOUDFLARE_TUNNEL_TOKEN=your_tunnel_token \
  bash
```

### Existing installation

The same bootstrap command is the supported manual upgrade path:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

Installer-managed checkouts must be clean. MyPaas fetches the configured upstream ref and synchronizes the checkout with the fetched revision, so rewritten/squashed `main` history does not require a Git merge.

---

## Automatic Self-Updates

Automatic updates are **opt-in**. After the current updater has been installed through bootstrap, enable the systemd timer with:

```bash
cd ~/MyPaas
AUTO_UPDATE_ENABLED=true \
AUTO_UPDATE_INTERVAL_MINUTES=30 \
bash scripts/configure-auto-update.sh
```

The policy is persisted in `/etc/mypaas/update.env`. The default interval is 30 minutes and the default tracked ref is `main`.

The updater does not blindly watch the mutable `latest` tag. For a new Git revision it waits for both API and dashboard images tagged with that exact commit SHA, then deploys the matching source/configuration and images together.

Useful commands:

```bash
# Check and apply an update immediately
cd ~/MyPaas
bash scripts/update-vm.sh

# Inspect the scheduled updater
systemctl status mypaas-update.timer

# View update logs
journalctl -u mypaas-update.service

# Disable automatic updates
AUTO_UPDATE_ENABLED=false bash scripts/configure-auto-update.sh
```

Updates refuse dirty Git checkouts and include best-effort runtime rollback if deployment or verification fails. Database migrations can be forward-only, so regular production backups are still required. See [ADR-018](docs/adr/ADR-018-automatic-self-update.md) for the design and trade-offs.

---

## Deployment Sources

### Git Repository

Use a repository when MyPaas should build the application from source.

Supported workflows include:

- Dockerfile deployment;
- Docker Compose multi-service deployment;
- static frontend deployment;
- repository root or selected base directory;
- Git webhook redeployment;
- Git revision deployment history and rollback.

Example base directories:

```text
apps/api
docs
frontend
services/worker
```

### Container Registry

Use a registry source when an image is already built by another CI/CD system.

Examples:

```text
nginx:latest
ghcr.io/example/my-api:v1.4.0
ghcr.io/example/my-api@sha256:<digest>
```

MyPaas pulls the configured public image, applies environment/resource settings, maps the application port, and routes traffic through the same project lifecycle used by built containers.

---

## Cloudflare Requirements

Before the dashboard and project routes can resolve publicly:

- the MyPaas domain must be active in Cloudflare DNS;
- the Cloudflare Tunnel must route the root MyPaas hostname and wildcard project hostname to Caddy (`HTTP` → `caddy:80`);
- Cloudflare DNS must contain proxied records for the root hostname and wildcard hostname pointing to `<tunnel-id>.cfargotunnel.com`.

A registrar transfer is not required. If the domain was purchased elsewhere, add it to Cloudflare and configure the Cloudflare nameservers at the registrar.

---

## Production Operations

Useful installer flags:

```bash
SKIP_DEPLOY=true bash scripts/install-vm.sh
FORCE_ENV=true bash scripts/install-vm.sh
SKIP_DOCKER_INSTALL=true bash scripts/install-vm.sh
INSTALL_WIZARD=true bash scripts/install-vm.sh
WIZARD_PUBLIC_TUNNEL=false INSTALL_WIZARD=true bash scripts/install-vm.sh
```

Deploy the current checkout:

```bash
bash scripts/deploy-to-vm.sh
```

Verify production:

```bash
bash scripts/verify-production.sh
RUN_BACKUP=true bash scripts/verify-production.sh
```

Inspect the control-plane containers:

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f
```

Production data directories are managed under `/var/lib/mypaas`; do not treat the Git checkout as the location of persistent application data.

---

## Local Development

### Prerequisites

- **Go 1.25.5** — matches `backend/go.mod`.
- **Node.js 22** — matches repository CI.
- **pnpm 10.22.0** — declared by `frontend/package.json`.
- **PostgreSQL 16**.
- **Docker + Docker Compose plugin**.
- **Caddy 2** when testing routing outside the Docker development stack.

### Setup

```bash
git clone https://github.com/nabilrn/MyPaas.git
cd MyPaas
cp .env.example .env
```

Start development dependencies and migrations:

```bash
make dev
```

Then run the API and dashboard in separate terminals:

```bash
make backend-dev
```

```bash
make frontend-dev
```

### Tests and build

```bash
make test
make lint
make build
```

Individual useful targets:

```bash
make test-backend
make test-frontend
make migrate-up
make migrate-down
make sqlc
make verify-prod
make help
```

Pull requests are also checked by GitHub Actions with backend tests, frontend unit/type/build checks, shell-script syntax checks, bootstrap regression tests, and production Compose rendering.

---

## Project Structure

```text
MyPaas/
├── backend/                  Go API and CLI
│   ├── cmd/                  API / CLI entry points
│   ├── internal/             Application and deployment services
│   ├── migrations/           PostgreSQL migrations
│   └── query/                sqlc queries
├── frontend/                 SvelteKit dashboard
├── docs/                     Architecture, PRD, and ADRs
├── scripts/                  Install, deploy, verify, and update tooling
├── docker-compose.dev.yml    Local dependencies
├── docker-compose.prod.yml   Production control plane
├── Caddyfile.*               Routing configuration
└── Makefile                  Development and operations targets
```

---

## Configuration

Start from `.env.example` for the complete supported configuration surface.

Core examples:

```bash
# Application / database
DATABASE_URL=postgres://user:pass@localhost:5432/mypaas_dev
ENVIRONMENT=development

# GitHub OAuth
GITHUB_CLIENT_ID=your_id
GITHUB_CLIENT_SECRET=your_secret

# Cloudflare
CLOUDFLARE_TUNNEL_TOKEN=your_token
CLOUDFLARE_ACCOUNT_ID=your_account_id

# Security
JWT_SECRET=your_256bit_secret_base64_encoded

# Docker
DOCKER_SOCKET=/var/run/docker.sock
```

Do not commit production `.env` files or generated secrets.

---

## Documentation

- **[Product Requirements](docs/PRD.md)** — product scope and behavior.
- **[Architecture](docs/ARCHITECTURE.md)** — technical design and system diagrams.
- **[Architecture Decisions](docs/adr/)** — documented design decisions, including self-update behavior.
- **[Conventions](CLAUDE.md)** — repository structure and coding conventions.
- **[Changelog](CHANGELOG.md)** — notable project changes.

---

## Contributing

1. Create a focused branch from the latest `main`.
2. Keep changes scoped to one bug or feature where practical.
3. Add or update regression tests.
4. Run the relevant backend/frontend/script checks locally.
5. Open a pull request with reproduction details, design decisions, migration impact, and validation results.
6. Keep generated code and documentation synchronized with source changes.

For production-sensitive changes, prefer fail-closed behavior at the backend boundary rather than relying on frontend validation alone.

---

## Troubleshooting

### Bootstrap reports a dirty checkout

Preserve, commit, or remove local Git changes before running the installer/updater. Automatic updates intentionally refuse to overwrite a modified checkout.

### Project domain does not resolve

Verify the Cloudflare Tunnel public hostname routes and the proxied root/wildcard DNS records.

### Production stack verification fails

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=200
bash scripts/verify-production.sh
```

### Local development database needs a reset

```bash
make docker-reset
make migrate-up
```

---

## License

MIT — see [LICENSE](LICENSE).

---

## Getting Help

- **Bug reports:** open a GitHub issue with reproduction steps and relevant logs.
- **Feature requests:** open an issue describing the workflow and expected behavior.
- **Documentation:** start with `docs/` and the architecture decision records.
- **Security issues:** use the repository's documented private security contact/process instead of publishing sensitive details in a public issue.

---

**Last updated:** 2026-08-10
