# Contributing to MyPaas

First off, thank you for considering contributing to MyPaas! It's people like you that make MyPaas such a great tool for self-hosted deployments.

## Ground Rules
- Be respectful and constructive.
- Always check if your issue or feature request already exists before opening a new one.
- MyPaas is built specifically for small-scale self-hosted setups (1 VM, minimal RAM footprint, single-node). Features proposing horizontal scaling (Kubernetes, Swarm) or very heavy tech stacks might not align with our goals. See `docs/PRD.md` and `docs/ARCHITECTURE.md` before submitting major features.

## Opening Issues

### Bug Reports
- Use the **Bug Report** template.
- Include a clear description of what went wrong.
- Provide step-by-step instructions to reproduce the issue.
- Include your server OS, MyPaas version, and any relevant logs (hide sensitive credentials like tokens or `.env` values).

### Feature Requests
- Use the **Feature Request** template.
- Describe the motivation: What problem are you trying to solve?
- Provide a clear, actionable suggested flow or MVP (Minimum Viable Product).
- Keep in mind MyPaas's core values: simplicity, low memory usage, and zero-configuration out-of-the-box experience.

## Submitting Pull Requests
1. Fork the repository and create your branch from `main`.
2. Ensure you follow our code conventions (detailed in `AGENTS.md`).
   - Go: Chi (not Gin/Echo), sqlc (not ORM), standard library testing.
   - Frontend: SvelteKit + Tailwind CSS + pnpm.
3. Update `CHANGELOG.md` with your changes.
4. Run all tests locally (`make test` or `pnpm test`).
5. Open a Pull Request and clearly describe what it fixes or adds.

If you have any questions, feel free to open a discussion or an issue!
