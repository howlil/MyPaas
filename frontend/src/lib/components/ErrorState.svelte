<script lang="ts">
	import { RotateCcw } from '@lucide/svelte';
	import { createEventDispatcher } from 'svelte';
	import ActionButton from './ActionButton.svelte';

	export let title = 'Could not load data';
	export let message = 'Something went wrong.';
	export let retryLabel = 'Retry';

	const dispatch = createEventDispatcher<{ retry: void }>();
</script>

<div class="p-4">
	<div class="alert-danger" role="alert">
		<div class="min-w-0">
			<p class="font-semibold">{title}</p>
			<p class="mt-1 max-w-2xl opacity-80">{message}</p>
			<div class="mt-3">
				<ActionButton variant="secondary" size="xs" on:click={() => dispatch('retry')}>
					<RotateCcw slot="icon" class="h-3.5 w-3.5" />
					{retryLabel}
				</ActionButton>
			</div>
		</div>
	</div>
</div>
