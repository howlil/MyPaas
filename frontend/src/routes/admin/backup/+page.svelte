<script lang="ts">
	import { onMount } from 'svelte';
	import { Cloud, Download, Database, HardDrive, ShieldCheck, LoaderCircle } from '@lucide/svelte';
	import { api } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';

	let loading = true;
	let savingS3 = false;

	let s3Config = {
		endpoint: '',
		bucket: '',
		region: '',
		access_key: '',
		secret_key: ''
	};

	onMount(() => {
		void loadConfig();
	});

	async function loadConfig() {
		loading = true;
		try {
			const data = await api.admin.getSettings();
			s3Config = {
				endpoint: ((data as any).s3_endpoint as string) || '',
				bucket: ((data as any).s3_bucket as string) || '',
				region: ((data as any).s3_region as string) || '',
				access_key: ((data as any).s3_access_key as string) || '',
				secret_key: ((data as any).s3_secret_key as string) || ''
			};
		} catch (error) {
			toast.error('Failed to load backup configuration');
			console.error(error);
		} finally {
			loading = false;
		}
	}

	async function saveS3Config() {
		savingS3 = true;
		try {
			await api.admin.updateS3Config(s3Config);
			toast.success('S3 configuration saved');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save S3 configuration');
		} finally {
			savingS3 = false;
		}
	}

	function downloadBackup() {
		window.location.href = '/api/admin/backup/download';
	}
</script>

<svelte:head>
	<title>Backup · MyPaas</title>
</svelte:head>

<div class="page-shell space-y-4 py-6">
	<p class="px-5 text-sm text-gray-500 dark:text-gray-400">Manage automated off-site backups and on-demand manual archives of PostgreSQL and platform configuration.</p>

	<SectionPanel title="How backup works" description="MyPaaS automatically creates and manages backups to ensure your platform state is safe." contentClass="p-0">
		<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 lg:grid-cols-3 lg:divide-x lg:divide-y-0">
			<div class="flex gap-3 p-4">
				<Database class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<div>
					<p class="text-sm font-medium text-gray-950 dark:text-white">1. Database Snapshot</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">The platform performs a consistent pg_dump of the entire PostgreSQL database, capturing users, projects, and deployment state.</p>
				</div>
			</div>
			<div class="flex gap-3 p-4">
				<ShieldCheck class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<div>
					<p class="text-sm font-medium text-gray-950 dark:text-white">2. Platform Config</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">Core platform files and configuration are archived alongside the database to ensure a seamless restoration process.</p>
				</div>
			</div>
			<div class="flex gap-3 p-4">
				<Cloud class="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden="true" />
				<div>
					<p class="text-sm font-medium text-gray-950 dark:text-white">3. Off-site Sync</p>
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">When configured, backups are automatically compressed, encrypted (if using HTTPS), and synced to an S3-compatible object storage provider daily.</p>
				</div>
			</div>
		</div>
	</SectionPanel>

	{#if loading}
		<div class="surface flex h-36 items-center justify-center">
			<LoaderCircle class="h-6 w-6 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
		</div>
	{:else}
		<SectionPanel title="S3 Automated Backup" description="Configure S3-compatible storage for automated daily backups.">
			<div class="mb-6 rounded-md border border-gray-200 bg-gray-50 p-4 dark:border-neutral-800 dark:bg-neutral-900">
				<h3 class="text-sm font-medium text-gray-900 dark:text-white">Cloudflare R2 Setup Guide</h3>
				<div class="mt-2 space-y-1.5 text-sm text-gray-600 dark:text-gray-400">
					<p>1. Go to Cloudflare Dashboard &rarr; <strong>R2 Object Storage</strong> and create a bucket.</p>
					<p>2. Click <strong>Manage R2 API Tokens</strong> and create a token with <strong>Object Read &amp; Write</strong> permissions.</p>
					<p>3. Copy the <strong>S3 Endpoint</strong> from the bucket settings (e.g., <code>https://&lt;account-id&gt;.r2.cloudflarestorage.com</code>).</p>
					<p>4. Use Region <code>auto</code> unless you specified a specific jurisdiction.</p>
				</div>
			</div>
			<div class="space-y-4 max-w-2xl">
				<label class="block">
					<span class="field-label">S3 Endpoint</span>
					<input type="text" bind:value={s3Config.endpoint} class="field" placeholder="https://s3.eu-central-1.amazonaws.com" />
				</label>
				<label class="block">
					<span class="field-label">Bucket</span>
					<input type="text" bind:value={s3Config.bucket} class="field" placeholder="mypaas-backups" />
				</label>
				<div class="grid grid-cols-2 gap-4">
					<label class="block">
						<span class="field-label">Region</span>
						<input type="text" bind:value={s3Config.region} class="field" placeholder="eu-central-1" />
					</label>
					<label class="block">
						<span class="field-label">Access Key</span>
						<input type="text" bind:value={s3Config.access_key} class="field" />
					</label>
				</div>
				<label class="block">
					<span class="field-label">Secret Key</span>
					<input type="password" bind:value={s3Config.secret_key} class="field" />
				</label>
			</div>
			<div class="mt-5 border-t border-gray-100 pt-4 dark:border-neutral-800">
				<ActionButton variant="primary" size="sm" loading={savingS3} on:click={saveS3Config}>Save S3 Config</ActionButton>
			</div>
		</SectionPanel>

		<SectionPanel title="Manual Backup" description="Download a complete snapshot of the database and platform configuration immediately.">
			<div class="flex items-center gap-4">
				<ActionButton variant="secondary" size="sm" on:click={downloadBackup}>
					<Download slot="icon" class="h-4 w-4" />
					Download Backup
				</ActionButton>
				<p class="text-sm text-gray-500 dark:text-gray-400">
					Note: This includes the database dump and core config, but does not include user project volumes.
				</p>
			</div>
		</SectionPanel>
	{/if}
</div>
