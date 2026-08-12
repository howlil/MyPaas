import { writable } from 'svelte/store';

/**
 * Navigation context published by deep routes so the global topbar can render
 * the full breadcrumb trail (e.g. Projects / cikiwir / Metrics). The topbar is
 * the single owner of route context; pages must not repeat the same title in
 * the body.
 */
export interface ShellContext {
	projectId?: string;
	projectName?: string;
}

export const shellContext = writable<ShellContext>({});

export function setShellContext(context: ShellContext) {
	shellContext.set(context);
}

export function clearShellContext() {
	shellContext.set({});
}
