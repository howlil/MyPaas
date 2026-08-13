<script lang="ts">
	export let label = 'Storage';
	export let value = 'Unavailable';
	export let detail = 'Host telemetry unavailable';
	export let percent = 0;
	export let className = '';

	$: usedPercent = Math.min(Math.max(Number.isFinite(percent) ? percent : 0, 0), 100);
	$: available = !/unavailable/i.test(`${value} ${detail}`);
	$: toneClass = usedPercent >= 90
		? 'bg-red-500'
		: usedPercent >= 80
			? 'bg-amber-500'
			: 'bg-gray-700 dark:bg-gray-300';
</script>

<article class={`flex h-full min-h-40 min-w-0 flex-col justify-between p-4 ${className}`.trim()} aria-label={`${label}: ${value}`}>
	<div>
		<div class="flex items-start justify-between gap-3">
			<p class="metric-label truncate">{label}</p>
			{#if available}
				<span class="metric-value shrink-0 text-xs font-medium text-gray-600 dark:text-gray-300">{usedPercent.toFixed(0)}%</span>
			{/if}
		</div>
		<p class="metric-value mt-2 truncate text-xl font-semibold tracking-tight text-gray-950 dark:text-white">{value}</p>
	</div>

	<div class="mt-5">
		{#if available}
			<div
				class="h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-neutral-800"
				role="progressbar"
				aria-label={`${label} used`}
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={Math.round(usedPercent)}
			>
				<div class={`h-full rounded-full transition-[width] duration-300 motion-reduce:transition-none ${toneClass}`} style={`width: ${usedPercent}%;`}></div>
			</div>
		{/if}
		<p class="mt-2 truncate text-xs text-gray-500 dark:text-gray-400">{detail}</p>
	</div>
</article>
