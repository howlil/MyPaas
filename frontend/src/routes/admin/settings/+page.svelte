<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Check, Copy, Download, Loader2, AlertTriangle, Package } from '@lucide/svelte';
	import { api, type MigrationStatus } from '$api';
	import { toast } from '$stores/toast';
	import PageHeader from '$components/PageHeader.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import ActionButton from '$components/ActionButton.svelte';

	let settings: Record<string, number> = {
		user_ram_quota_gb: 0,
		user_cpu_quota: 0,
		max_projects: 0,
		max_concurrent_deploys: 0,
		project_default_ram_mb: 0,
		project_default_cpu: 0,
		build_timeout_minutes: 0
	};
	let loadingSettings = true;
	let savingSettings = false;

	let migration: MigrationStatus | null = null;
	let preparingMigration = false;
	let pollingInterval: ReturnType<typeof setInterval>;
	let copiedText: string | null = null;

	const settingsConfig = [
		{ key: 'user_ram_quota_gb', label: 'User RAM Quota (GB)' },
		{ key: 'user_cpu_quota', label: 'User CPU Quota (cores)' },
		{ key: 'max_projects', label: 'Max Projects' },
		{ key: 'max_concurrent_deploys', label: 'Max Concurrent Deploys' },
		{ key: 'project_default_ram_mb', label: 'Default Project RAM (MB)' },
		{ key: 'project_default_cpu', label: 'Default Project CPU (cores)' },
		{ key: 'build_timeout_minutes', label: 'Build Timeout (minutes)' }
	];

	onMount(async () => {
		await loadSettings();
	});

	onDestroy(() => {
		if (pollingInterval) clearInterval(pollingInterval);
	});

	async function loadSettings() {
		try {
			const data = await api.admin.getSettings();
			settings = { ...settings, ...data };
		} catch (error) {
			toast.error('Failed to load settings');
			console.error(error);
		} finally {
			loadingSettings = false;
		}
	}

	async function saveSettings() {
		if (savingSettings) return;
		savingSettings = true;
		try {
			const updated = await api.admin.updateSettings(settings);
			settings = { ...settings, ...updated };
			toast.success('Settings saved successfully');
		} catch (error) {
			toast.error('Failed to save settings');
			console.error(error);
		} finally {
			savingSettings = false;
		}
	}

	async function startMigration() {
		if (preparingMigration) return;
		preparingMigration = true;
		try {
			migration = await api.admin.prepareMigration();
			if (migration.status === 'preparing') {
				startPolling();
			}
		} catch (error) {
			toast.error('Failed to prepare migration');
			console.error(error);
			preparingMigration = false;
		}
	}

	function startPolling() {
		if (pollingInterval) clearInterval(pollingInterval);
		pollingInterval = setInterval(async () => {
			if (!migration?.id) return;
			try {
				const status = await api.admin.migrationStatus(migration.id);
				migration = status;
				if (status.status !== 'preparing') {
					clearInterval(pollingInterval);
					preparingMigration = false;
					if (status.status === 'failed') {
						toast.error(status.error || 'Migration preparation failed');
					} else if (status.status === 'ready') {
						toast.success('Migration package is ready');
					}
				}
			} catch (error) {
				console.error('Error polling migration status:', error);
			}
		}, 3000);
	}

	async function copyToClipboard(text: string, id: string) {
		try {
			await navigator.clipboard.writeText(text);
			copiedText = id;
			toast.success('Copied!');
			setTimeout(() => {
				if (copiedText === id) copiedText = null;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy', err);
			toast.error('Failed to copy text');
		}
	}

	function formatBytes(bytes?: number) {
		if (!bytes) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function formatHoursLeft(expiresAt?: string) {
		if (!expiresAt) return 0;
		const diff = new Date(expiresAt).getTime() - Date.now();
		return Math.max(0, Math.floor(diff / (1000 * 60 * 60)));
	}

	$: downloadUrl = migration?.downloadToken ? `/api/admin/migrate/${migration.id}/download?token=${migration.downloadToken}` : '#';
</script>

<svelte:head>
	<title>Admin Settings - MyPaas</title>
</svelte:head>

<div class="page-shell max-w-5xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
	<PageHeader title="Platform Settings" description="Manage platform configurations and migrations" />

	<SectionPanel title="Resource Configuration" description="Configure default platform limits and resource quotas." className="mb-8">
		{#if loadingSettings}
			<div class="flex h-32 items-center justify-center">
				<Loader2 class="h-6 w-6 animate-spin text-brand-600" />
			</div>
		{:else}
			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
				{#each settingsConfig as { key, label }}
					<div class="space-y-1.5">
						<label for={key} class="block text-sm font-medium text-gray-700 dark:text-gray-300">
							{label}
						</label>
						<input
							type="number"
							id={key}
							bind:value={settings[key]}
							class="field block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-brand-500 focus:outline-none focus:ring-1 focus:ring-brand-500 dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:focus:border-brand-500 dark:focus:ring-brand-500"
						/>
					</div>
				{/each}
			</div>
		{/if}

		<svelte:fragment slot="actions">
			<ActionButton variant="primary" loading={savingSettings} on:click={saveSettings} disabled={loadingSettings}>
				Save Changes
			</ActionButton>
		</svelte:fragment>
	</SectionPanel>

	<SectionPanel title="VM Migration" description="Migrate your entire MyPaas installation to a new server.">
		{#if !migration || (migration.status === 'failed' && !preparingMigration)}
			<div class="space-y-6">
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Generate a migration package containing your complete MyPaas state — database, project volumes, configurations, and encrypted secrets. Transfer this package to your new VM to restore everything.
				</p>

				<div class="rounded-md border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900/50 dark:bg-yellow-900/20">
					<div class="flex gap-3">
						<AlertTriangle class="h-5 w-5 shrink-0 text-yellow-600 dark:text-yellow-500" />
						<div class="text-sm text-yellow-800 dark:text-yellow-200">
							<strong>Warning:</strong> All running project containers will be temporarily stopped during export. They will restart automatically after the package is created.
						</div>
					</div>
				</div>

				<ActionButton variant="primary" size="md" on:click={startMigration}>
					<Package class="mr-2 h-4 w-4" />
					Prepare Migration Package
				</ActionButton>
			</div>
		{:else if migration.status === 'preparing' || preparingMigration}
			<div class="flex flex-col items-center justify-center space-y-4 rounded-lg border border-gray-200 py-12 dark:border-gray-800">
				<div class="relative flex h-16 w-16 items-center justify-center rounded-full bg-brand-100 dark:bg-brand-900/30">
					<div class="absolute inset-0 animate-ping rounded-full bg-brand-400 opacity-20"></div>
					<Loader2 class="h-8 w-8 animate-spin text-brand-600 dark:text-brand-500" />
				</div>
				<p class="text-center text-sm font-medium text-gray-900 dark:text-gray-100">
					Creating migration package...
				</p>
				<p class="text-center text-sm text-gray-500 dark:text-gray-400">
					This may take a few minutes depending on data size.
				</p>
			</div>
		{:else if migration.status === 'ready'}
			<div class="space-y-8">
				<div class="rounded-md border border-green-200 bg-green-50 p-5 dark:border-green-900/30 dark:bg-green-900/10">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
						<div>
							<h3 class="text-sm font-medium text-green-800 dark:text-green-400">Migration package ready</h3>
							<p class="mt-1 text-sm text-green-700 dark:text-green-500">
								This package expires in {formatHoursLeft(migration.expiresAt)} hours
							</p>
						</div>
						<a
							href={downloadUrl}
							class="inline-flex items-center justify-center gap-2 rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900"
						>
							<Download class="h-4 w-4" />
							Download Package ({formatBytes(migration.sizeBytes)})
						</a>
					</div>
				</div>

				<div>
					<h3 class="mb-6 text-lg font-medium text-gray-900 dark:text-white">Migration Guide</h3>
					<div class="space-y-6">
						<!-- Step 1 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								1
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Download the package</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">Click the download button above to save the backup to your local machine.</p>
							</div>
						</div>

						<!-- Step 2 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								2
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Transfer to new VM</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">Copy the downloaded file to your new server's <code>/tmp</code> directory.</p>
								<div class="group relative mt-2 rounded-md bg-gray-900 p-3 pr-12 text-sm text-gray-300">
									<code class="block overflow-x-auto whitespace-nowrap font-mono">scp mypaas-export-*.tar.gz user@new-vm:/tmp/</code>
									<button
										on:click={() => copyToClipboard('scp mypaas-export-*.tar.gz user@new-vm:/tmp/', 'step2')}
										class="absolute right-2 top-2 rounded p-1.5 text-gray-400 hover:bg-gray-800 hover:text-white focus:outline-none"
										aria-label="Copy code"
									>
										{#if copiedText === 'step2'}
											<Check class="h-4 w-4 text-green-400" />
										{:else}
											<Copy class="h-4 w-4" />
										{/if}
									</button>
								</div>
							</div>
						</div>

						<!-- Step 3 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								3
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Install prerequisites</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">SSH into your new VM and install Docker and Git.</p>
								<div class="group relative mt-2 rounded-md bg-gray-900 p-3 pr-12 text-sm text-gray-300">
									<code class="block overflow-x-auto whitespace-nowrap font-mono">sudo apt update && sudo apt install -y docker.io git<br/>sudo systemctl enable --now docker</code>
									<button
										on:click={() => copyToClipboard('sudo apt update && sudo apt install -y docker.io git\nsudo systemctl enable --now docker', 'step3')}
										class="absolute right-2 top-2 rounded p-1.5 text-gray-400 hover:bg-gray-800 hover:text-white focus:outline-none"
										aria-label="Copy code"
									>
										{#if copiedText === 'step3'}
											<Check class="h-4 w-4 text-green-400" />
										{:else}
											<Copy class="h-4 w-4" />
										{/if}
									</button>
								</div>
							</div>
						</div>

						<!-- Step 4 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								4
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Clone MyPaas</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">Clone the repository on the new VM.</p>
								<div class="group relative mt-2 rounded-md bg-gray-900 p-3 pr-12 text-sm text-gray-300">
									<code class="block overflow-x-auto whitespace-nowrap font-mono">git clone https://github.com/your-org/mypaas.git mypaas && cd mypaas</code>
									<button
										on:click={() => copyToClipboard('git clone https://github.com/your-org/mypaas.git mypaas && cd mypaas', 'step4')}
										class="absolute right-2 top-2 rounded p-1.5 text-gray-400 hover:bg-gray-800 hover:text-white focus:outline-none"
										aria-label="Copy code"
									>
										{#if copiedText === 'step4'}
											<Check class="h-4 w-4 text-green-400" />
										{:else}
											<Copy class="h-4 w-4" />
										{/if}
									</button>
								</div>
							</div>
						</div>

						<!-- Step 5 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								5
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Run import</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">Execute the migration import script.</p>
								<div class="group relative mt-2 rounded-md bg-gray-900 p-3 pr-12 text-sm text-gray-300">
									<code class="block overflow-x-auto whitespace-nowrap font-mono">bash scripts/migrate-import.sh /tmp/mypaas-export-*.tar.gz</code>
									<button
										on:click={() => copyToClipboard('bash scripts/migrate-import.sh /tmp/mypaas-export-*.tar.gz', 'step5')}
										class="absolute right-2 top-2 rounded p-1.5 text-gray-400 hover:bg-gray-800 hover:text-white focus:outline-none"
										aria-label="Copy code"
									>
										{#if copiedText === 'step5'}
											<Check class="h-4 w-4 text-green-400" />
										{:else}
											<Copy class="h-4 w-4" />
										{/if}
									</button>
								</div>
							</div>
						</div>

						<!-- Step 6 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								6
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Redeploy projects</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">Trigger a redeploy of all your projects on the new infrastructure.</p>
								<div class="group relative mt-2 rounded-md bg-gray-900 p-3 pr-12 text-sm text-gray-300">
									<code class="block overflow-x-auto whitespace-nowrap font-mono">for p in $(docker ps -aq --filter label=mypaas.project); do docker restart $p; done</code>
									<button
										on:click={() => copyToClipboard('for p in $(docker ps -aq --filter label=mypaas.project); do docker restart $p; done', 'step6')}
										class="absolute right-2 top-2 rounded p-1.5 text-gray-400 hover:bg-gray-800 hover:text-white focus:outline-none"
										aria-label="Copy code"
									>
										{#if copiedText === 'step6'}
											<Check class="h-4 w-4 text-green-400" />
										{:else}
											<Copy class="h-4 w-4" />
										{/if}
									</button>
								</div>
							</div>
						</div>

						<!-- Step 7 -->
						<div class="relative flex gap-4 rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-800 dark:bg-gray-950">
							<div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand-100 font-bold text-brand-700 dark:bg-brand-900/40 dark:text-brand-400">
								7
							</div>
							<div class="flex-1 space-y-2">
								<h4 class="text-base font-semibold text-gray-900 dark:text-white">Verify</h4>
								<p class="text-sm text-gray-600 dark:text-gray-400">Update your DNS records to point to the new VM IP, then check your dashboard and verify all project subdomains are accessible.</p>
							</div>
						</div>

					</div>
				</div>
			</div>
		{/if}
	</SectionPanel>
</div>
