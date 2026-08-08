<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { api } from '$api';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { ExternalLink } from '@lucide/svelte';

	let token = '';
	let zoneId = '';
	let loading = false;
	let error = '';

	const dispatch = createEventDispatcher();

	async function submit() {
		if (!token.trim() || !zoneId.trim()) {
			error = 'Token and Zone ID are required';
			return;
		}

		loading = true;
		error = '';

		try {
			await api.admin.updateCloudflareConfig(token.trim(), zoneId.trim());
			dispatch('success');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save configuration';
		} finally {
			loading = false;
		}
	}
</script>

<SectionPanel
	title="Cloudflare Analytics Setup"
	description="Configure Cloudflare to view global traffic, bandwidth, and errors for your projects."
>
	<form on:submit|preventDefault={submit} class="space-y-4">
		{#if error}
			<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-400">
				{error}
			</div>
		{/if}

		<div class="space-y-3 rounded-md bg-blue-50 p-4 text-sm text-blue-800 dark:bg-blue-950/40 dark:text-blue-300">
			<h4 class="font-medium text-blue-900 dark:text-blue-200">How to get your credentials</h4>
			<ol class="list-inside list-decimal space-y-1">
				<li>Go to your Cloudflare Dashboard and select your domain.</li>
				<li>Scroll down on the <strong>Overview</strong> page to find your <strong>Zone ID</strong> on the right sidebar.</li>
				<li>Click your profile icon &gt; <strong>My Profile</strong> &gt; <strong>API Tokens</strong>.</li>
				<li>Click <strong>Create Token</strong> &gt; <strong>Create Custom Token</strong>.</li>
				<li>Set Permissions: <code>Zone</code> &gt; <code>Analytics</code> &gt; <code>Read</code>.</li>
				<li>Set Zone Resources: <code>Include</code> &gt; <code>Specific Zone</code> &gt; Select your domain.</li>
			</ol>
			<div class="mt-2">
				<a href="https://dash.cloudflare.com/profile/api-tokens" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 font-medium hover:underline">
					Go to API Tokens <ExternalLink class="h-3 w-3" />
				</a>
			</div>
		</div>

		<div>
			<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="zone-id">Zone ID</label>
			<input id="zone-id" type="text" bind:value={zoneId} placeholder="e.g. 023e105f4ecef8ad9ca31a8372d0c353" required class="field w-full" />
		</div>

		<div>
			<label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300" for="api-token">API Token</label>
			<input id="api-token" type="password" bind:value={token} placeholder="e.g. Y7_abcdefghijklmnopqrstuvwxyz1234567890" required class="field w-full font-mono" />
		</div>

		<div class="pt-2">
			<ActionButton type="submit" variant="primary" {loading}>Save Configuration</ActionButton>
		</div>
	</form>
</SectionPanel>
