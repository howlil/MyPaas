<script lang="ts">
	export let totalBytes = 0;
	export let availableBytes = 0;
	export let className = '';

	$: usedBytes = Math.max(0, totalBytes - availableBytes);
	$: usedPercent = totalBytes > 0 ? Math.min(Math.max((usedBytes / totalBytes) * 100, 0), 100) : 0;
	$: toneClass = usedPercent >= 90
		? 'bg-red-500'
		: usedPercent >= 80
			? 'bg-amber-500'
			: 'bg-gray-700 dark:bg-gray-300';

	function formatBytes(value: number) {
		if (!Number.isFinite(value) || value < 0) return '-';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let amount = value;
		let index = 0;
		while (amount >= 1024 && index < units.length - 1) {
			amount /= 1024;
			index += 1;
		}
		const digits = amount >= 100 || index === 0 ? 0 : amount >= 10 ? 1 : 2;
		return `${amount.toFixed(digits)} ${units[index]}`;
	}
</script>

<div class={`flex h-full min-h-40 flex-col justify-between p-4 ${className}`}>
	<div>
		<div class="flex items-start justify-between gap-3">
			<p class="metric-label">Storage</p>
			{#if totalBytes > 0}
				<span class="metric-value text-xs font-medium text-gray-600 dark:text-gray-300">{usedPercent.toFixed(0)}%</span>
			{/if}
		</div>
		<p class="metric-value mt-2 text-xl font-semibold tracking-tight text-gray-950 dark:text-white">
			{totalBytes > 0 ? `${formatBytes(usedBytes)} used` : 'Unavailable'}
		</p>
	</div>

	{#if totalBytes > 0}
		<div class="mt-5">
			<div
				class="h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800"
				role="progressbar"
				aria-label="Host storage used"
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={Math.round(usedPercent)}
			>
				<div class={`h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none ${toneClass}`} style={`width: ${usedPercent}%;`}></div>
			</div>
			<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{formatBytes(availableBytes)} available · {formatBytes(totalBytes)} total</p>
		</div>
	{:else}
		<p class="mt-5 text-xs text-gray-500 dark:text-gray-400">Host storage telemetry unavailable</p>
	{/if}
</div>
