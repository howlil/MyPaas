---
name: mypaas-svelte
description: "Use when writing or refactoring SvelteKit and TypeScript code in the MyPaas frontend to ensure strict typing, consistent architecture, and clean UI components."
---

# MyPaas Frontend (SvelteKit) Professional Standard

This skill defines the professional standards and architecture rules for all frontend code written in the MyPaas UI. ALWAYS follow these guidelines when creating new components, pages, or API clients.

## 1. Core Stack
- **Framework:** SvelteKit (latest stable). Do not suggest React, Vue, Next.js, or Nuxt.
- **Language:** TypeScript in strict mode. Avoid `any` at all costs.
- **Styling:** Tailwind CSS. Use utility classes over custom `<style>` blocks unless absolutely necessary.
- **Package Manager:** `pnpm`. NEVER use `npm` or `yarn`.

## 2. Naming Conventions
- **Components:** `PascalCase.svelte` (e.g., `DeployButton.svelte`, `ProjectCard.svelte`).
- **Functions & Variables:** `camelCase` (e.g., `fetchDeployments`, `isMenuOpen`).
- **Types & Interfaces:** `PascalCase` (e.g., `DeploymentStatus`, `ProjectDetail`).
- **Constants:** `SCREAMING_SNAKE_CASE` (e.g., `MAX_RETRY_COUNT`, `DEFAULT_APP_PORT`).

## 3. Component Architecture
- **Keep it Dumb:** Prefer "dumb" UI components in `src/lib/components/` that receive data via `export let` (props) and emit actions via `createEventDispatcher`.
- **Smart Pages:** Route files (`+page.svelte`) should act as the "smart" containers. They fetch data (or read from `+page.ts`), handle API calls, and pass state down to dumb components.
- **Stores:** Use Svelte stores (`src/lib/stores/`) for global state (e.g., active user session, theme toggle, toasts).

## 4. API & Error Handling
- **API Client:** All API calls must go through the centralized client in `src/lib/api/`. Do not use raw `fetch()` inside `.svelte` files.
- **Errors as Data:** Do not use `try/catch` exceptions to control business flow. The API client must return `{ data, error }`.
  ```typescript
  // GOOD
  const { data, error } = await api.projects.get(id);
  if (error) {
      toast.error(error.message);
      return;
  }
  ```
- **Domain Errors:** Display friendly error messages based on the `error.code` returned by the Go backend (e.g., translating `DOCKERFILE_NOT_FOUND` into a human-readable UI warning).

## 5. Streaming & Realtime
- **Server-Sent Events (SSE):** MyPaas uses SSE for all realtime logs and metrics. DO NOT implement WebSockets. 
- **Cleanup:** Always close `EventSource` connections in the Svelte `onDestroy` lifecycle hook to prevent memory leaks and zombie connections.

## 6. Styling & UI/UX
- **Tailwind First:** Build layouts with Flexbox and CSS Grid via Tailwind classes.
- **Dark Mode:** Ensure all components support both light and dark mode using Tailwind's `dark:` variant.
- **Animations:** Use Svelte's built-in `transition:` and `animate:` directives (e.g., `fade`, `slide`, `fly`) for smooth micro-interactions. Avoid heavy CSS keyframe animations unless necessary.
- **Charts:** Use `Chart.js` (via wrapper if necessary) for rendering container metrics (CPU/RAM).
