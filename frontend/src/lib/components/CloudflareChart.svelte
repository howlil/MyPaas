<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Chart, type ChartConfiguration, registerables } from 'chart.js';
	import type { TimeseriesDataPoint } from '$types';

	Chart.register(...registerables);

	export let data: TimeseriesDataPoint[] = [];

	let canvas: HTMLCanvasElement;
	let chart: Chart | null = null;

	$: labels = data.map(d => {
		const date = new Date(d.timestamp);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	});
	
	$: requestsData = data.map(d => d.requests);
	$: bandwidthData = data.map(d => Number((d.bandwidth / (1024 * 1024)).toFixed(2))); // MB

	const config = (): ChartConfiguration => ({
		type: 'line',
		data: {
			labels,
			datasets: [
				{
					label: 'Requests',
					data: requestsData,
					borderColor: '#3b82f6', // blue-500
					backgroundColor: '#3b82f620',
					borderWidth: 2,
					pointRadius: 0,
					pointHoverRadius: 4,
					tension: 0.4,
					fill: true,
					yAxisID: 'y'
				},
				{
					label: 'Bandwidth (MB)',
					data: bandwidthData,
					borderColor: '#10b981', // emerald-500
					backgroundColor: '#10b98120',
					borderWidth: 2,
					pointRadius: 0,
					pointHoverRadius: 4,
					tension: 0.4,
					fill: true,
					yAxisID: 'y1'
				}
			]
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			interaction: {
				mode: 'index',
				intersect: false,
			},
			plugins: {
				legend: { 
					display: true,
					position: 'top',
					labels: {
						usePointStyle: true,
						boxWidth: 6
					}
				},
				tooltip: {
					backgroundColor: 'rgba(17, 24, 39, 0.9)',
					titleColor: '#fff',
					bodyColor: '#e5e7eb',
					borderColor: 'rgba(255,255,255,0.1)',
					borderWidth: 1,
					padding: 10,
					boxPadding: 4,
					usePointStyle: true
				}
			},
			scales: {
				x: {
					grid: { display: true, color: 'rgba(156, 163, 175, 0.05)', drawTicks: false },
					border: { display: false },
					ticks: { color: '#9ca3af', font: { size: 11 }, maxTicksLimit: 8 }
				},
				y: {
					type: 'linear',
					display: true,
					position: 'left',
					beginAtZero: true,
					grid: { color: 'rgba(156, 163, 175, 0.05)', drawTicks: false },
					border: { display: false },
					ticks: {
						color: '#9ca3af',
						font: { size: 11 }
					}
				},
				y1: {
					type: 'linear',
					display: true,
					position: 'right',
					beginAtZero: true,
					grid: { display: false },
					border: { display: false },
					ticks: {
						color: '#9ca3af',
						font: { size: 11 },
						callback: (v) => `${v} MB`
					}
				}
			}
		}
	});

	onMount(() => {
		chart = new Chart(canvas, config());
	});

	onDestroy(() => chart?.destroy());

	$: if (chart && data.length > 0) {
		chart.data.labels = labels;
		chart.data.datasets[0].data = requestsData;
		chart.data.datasets[1].data = bandwidthData;
		chart.update();
	}
</script>

<div class="h-64 w-full relative">
	<canvas bind:this={canvas}></canvas>
</div>
