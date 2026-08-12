<script lang="ts">
	import type { ProjectStatus, DeployStatus } from '$types';

	export let status: ProjectStatus | DeployStatus;
	export let pulse = false;
	export let label: string | undefined = undefined;
	/**
	 * Default rendering is a semantic dot with neutral text. A tinted pill is
	 * reserved for states that genuinely need emphasis (failed / crashed), or
	 * when a callsite explicitly requests it.
	 */
	export let emphasis: 'auto' | 'always' | 'never' = 'auto';

	type StatusConfig = { label: string; dot: string; pill: string };

	const neutralPill = 'border-gray-200 bg-gray-50 text-gray-600 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400';
	const cfg: Record<string, StatusConfig> = {
		running:     { label: 'Running',     dot: 'bg-emerald-500', pill: 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200' },
		stopped:     { label: 'Stopped',     dot: 'bg-gray-400 dark:bg-gray-500', pill: neutralPill },
		crashed:     { label: 'Crashed',     dot: 'bg-red-500',     pill: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200' },
		building:    { label: 'Building',    dot: 'bg-amber-500',   pill: 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200' },
		pending:     { label: 'Pending',     dot: 'bg-sky-500',     pill: 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-200' },
		queued:      { label: 'Queued',      dot: 'bg-sky-500',     pill: 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-200' },
		cloning:     { label: 'Cloning',     dot: 'bg-amber-500',   pill: 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200' },
		starting:    { label: 'Starting',    dot: 'bg-amber-500',   pill: 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200' },
		failed:      { label: 'Failed',      dot: 'bg-red-500',     pill: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200' },
		rolled_back: { label: 'Rolled back', dot: 'bg-gray-400 dark:bg-gray-500', pill: neutralPill }
	};

	$: c = cfg[status] ?? { label: status, dot: 'bg-gray-400 dark:bg-gray-500', pill: neutralPill };
	$: isPulsing = pulse && ['building', 'cloning', 'starting', 'queued'].includes(status);
	$: emphasized = emphasis === 'always' || (emphasis === 'auto' && (status === 'crashed' || status === 'failed'));
</script>

{#if emphasized}
	<span class="inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium leading-5 {c.pill}">
		{#if isPulsing}
			<span class="relative flex h-1.5 w-1.5">
				<span class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 {c.dot}"></span>
				<span class="status-dot relative {c.dot}"></span>
			</span>
		{:else}
			<span class="status-dot {c.dot}"></span>
		{/if}
		{label ?? c.label}
	</span>
{:else}
	<span class="inline-flex items-center gap-1.5 text-xs font-medium text-gray-600 dark:text-gray-300">
		{#if isPulsing}
			<span class="relative flex h-1.5 w-1.5">
				<span class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 {c.dot}"></span>
				<span class="status-dot relative {c.dot}"></span>
			</span>
		{:else}
			<span class="status-dot {c.dot}"></span>
		{/if}
		{label ?? c.label}
	</span>
{/if}
