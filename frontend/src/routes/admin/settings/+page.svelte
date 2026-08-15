<script lang="ts">
	import { onMount } from 'svelte';
	import { LoaderCircle } from '@lucide/svelte';
	import { api, type HostStats } from '$api';
	import { toast } from '$stores/toast';
	import ActionButton from '$components/ActionButton.svelte';
	import SectionPanel from '$components/SectionPanel.svelte';
	import { goto } from '$app/navigation';

	type SettingKey = 'user_ram_quota_gb' | 'user_cpu_quota' | 'max_projects' | 'build_timeout_minutes';
	type NumericSettings = Record<SettingKey, number>;

	const defaultSettings: NumericSettings = {
		user_ram_quota_gb: 0,
		user_cpu_quota: 0,
		max_projects: 0,
		build_timeout_minutes: 0
	};

	let settings: NumericSettings = { ...defaultSettings };
	let savedSettings: NumericSettings = { ...defaultSettings };
	let hostStats: HostStats | null = null;
	let loadingSettings = true;
	let savingSettings = false;
	let savingS3 = false;
	let triggeringBackup = false;
	let triggeringUpdate = false;
	let updateOverlayOpen = false;

	let s3Config = {
		endpoint: '',
		bucket: '',
		region: '',
		access_key: '',
		secret_key: ''
	};

	$: settingsChanged = (Object.keys(defaultSettings) as SettingKey[]).some((key) => settings[key] !== savedSettings[key]);
	$: validationErrors = {
		user_ram_quota_gb: numberError(settings.user_ram_quota_gb, 0, 1024, false, 'RAM quota must be greater than 0 and at most 1024 GB.'),
		user_cpu_quota: numberError(settings.user_cpu_quota, 0, 256, false, 'CPU quota must be greater than 0 and at most 256 cores.'),
		max_projects: numberError(settings.max_projects, 1, 10000, true, 'Maximum projects must be a whole number between 1 and 10000.'),
		build_timeout_minutes: numberError(settings.build_timeout_minutes, 1, 1440, true, 'Build timeout must be a whole number between 1 and 1440 minutes.')
	};
	$: hasValidationErrors = Object.values(validationErrors).some(Boolean);
	$: hostMemoryTotal = hostStats?.memory?.total_bytes ?? hostStats?.host_ram_bytes ?? 0;
	$: hostMemoryUsed = hostStats?.memory ? Math.max(0, hostStats.memory.total_bytes - hostStats.memory.available_bytes) : 0;
	$: hostStorageUsed = hostStats?.storage ? Math.max(0, hostStats.storage.total_bytes - hostStats.storage.available_bytes) : 0;

	onMount(() => {
		void loadSettings();
	});

	async function loadSettings() {
		loadingSettings = true;
		try {
			const [data, capacity] = await Promise.all([
				api.admin.getSettings(),
				api.admin.getHostStats().catch(() => null)
			]);
			settings = {
				user_ram_quota_gb: numericValue(data.user_ram_quota_gb),
				user_cpu_quota: numericValue(data.user_cpu_quota),
				max_projects: numericValue(data.max_projects),
				build_timeout_minutes: numericValue(data.build_timeout_minutes)
			};
			s3Config = {
				endpoint: ((data as any).s3_endpoint as string) || '',
				bucket: ((data as any).s3_bucket as string) || '',
				region: ((data as any).s3_region as string) || '',
				access_key: ((data as any).s3_access_key as string) || '',
				secret_key: ((data as any).s3_secret_key as string) || ''
			};
			savedSettings = { ...settings };
			hostStats = capacity;
		} catch (error) {
			toast.error('Failed to load settings');
			console.error(error);
		} finally {
			loadingSettings = false;
		}
	}

	async function saveSettings() {
		if (savingSettings || !settingsChanged || hasValidationErrors) return;
		savingSettings = true;
		try {
			const updated = await api.admin.updateSettings(settings);
			settings = {
				user_ram_quota_gb: numericValue(updated.user_ram_quota_gb, settings.user_ram_quota_gb),
				user_cpu_quota: numericValue(updated.user_cpu_quota, settings.user_cpu_quota),
				max_projects: numericValue(updated.max_projects, settings.max_projects),
				build_timeout_minutes: numericValue(updated.build_timeout_minutes, settings.build_timeout_minutes)
			};
			savedSettings = { ...settings };
			toast.success('Platform settings saved');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to save settings');
			console.error(error);
		} finally {
			savingSettings = false;
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

	async function triggerBackup() {
		triggeringBackup = true;
		try {
			await api.admin.triggerBackup();
			toast.success('Backup triggered successfully');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to trigger backup');
		} finally {
			triggeringBackup = false;
		}
	}

	async function triggerUpdate() {
		triggeringUpdate = true;
		try {
			await api.admin.triggerUpdate();
			updateOverlayOpen = true;
			startUpdatePolling();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : 'Failed to trigger update');
			console.error(error);
		} finally {
			triggeringUpdate = false;
		}
	}

	function startUpdatePolling() {
		let wasDown = false;
		const poll = setInterval(async () => {
			try {
				const res = await fetch('/api/health');
				if (res.ok) {
					if (wasDown) {
						clearInterval(poll);
						window.location.href = '/';
					}
				} else {
					wasDown = true;
				}
			} catch {
				wasDown = true;
			}
		}, 3000);
	}

	function discardChanges() {
		settings = { ...savedSettings };
	}

	function numericValue(value: unknown, fallback = 0) {
		return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
	}

	function numberError(value: number, min: number, max: number, integer: boolean, message: string) {
		if (!Number.isFinite(value) || value < min || value > max || (min === 0 && value === 0) || (integer && !Number.isInteger(value))) return message;
		return '';
	}

	function formatBytes(value: number) {
		if (!Number.isFinite(value) || value <= 0) return 'Unavailable';
		const units = ['B', 'KB', 'MB', 'GB', 'TB'];
		let amount = value;
		let index = 0;
		while (amount >= 1024 && index < units.length - 1) {
			amount /= 1024;
			index += 1;
		}
		return `${amount.toFixed(amount >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
	}
</script>

<svelte:head>
	<title>Settings · MyPaas</title>
</svelte:head>

{#if updateOverlayOpen}
	<div class="fixed inset-0 z-50 flex flex-col items-center justify-center bg-white/90 backdrop-blur-sm dark:bg-gray-950/90">
		<LoaderCircle class="mb-4 h-12 w-12 animate-spin text-gray-500 dark:text-gray-400" />
		<h2 class="text-xl font-medium text-gray-900 dark:text-white">Updating MyPaas</h2>
		<p class="mt-2 text-sm text-gray-500 dark:text-gray-400">Please wait while the platform restarts...</p>
	</div>
{/if}

<div class="page-shell space-y-4 py-6">
	<p class="px-5 text-sm text-gray-500 dark:text-gray-400">Configure the guardrails enforced by this MyPaas control plane. Capacity context is shown so limits are not edited as isolated numbers.</p>

	<SectionPanel title="Platform capacity" description="Current host capacity and project allocation context. This is context for configuration, not another telemetry dashboard." contentClass="p-0">
		{#if hostStats}
			<div class="grid divide-y divide-gray-100 dark:divide-neutral-800 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
				<div class="p-4">
					<p class="metric-label">Memory</p>
					<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{formatBytes(hostMemoryTotal)}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{formatBytes(hostStats.allocated_ram_mb * 1024 * 1024)} allocated{hostStats.memory ? ` · ${formatBytes(hostMemoryUsed)} host used` : ''}</p>
				</div>
				<div class="p-4">
					<p class="metric-label">CPU</p>
					<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{hostStats.host_cpu_cores} core{hostStats.host_cpu_cores === 1 ? '' : 's'}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{hostStats.allocated_cpu.toFixed(2)} cores allocated</p>
				</div>
				<div class="p-4">
					<p class="metric-label">Storage</p>
					<p class="metric-value mt-1 text-xl font-semibold text-gray-950 dark:text-white">{hostStats.storage ? formatBytes(hostStats.storage.total_bytes) : 'Unavailable'}</p>
					<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{hostStats.storage ? `${formatBytes(hostStorageUsed)} used · ${formatBytes(hostStats.storage.available_bytes)} available` : 'Host storage telemetry is unavailable'}</p>
				</div>
			</div>
		{:else}
			<p class="p-4 text-sm text-gray-500 dark:text-gray-400">Host capacity context is unavailable. Platform limits can still be edited below.</p>
		{/if}
	</SectionPanel>

	{#if loadingSettings}
		<div class="surface flex h-36 items-center justify-center">
			<LoaderCircle class="h-6 w-6 animate-spin motion-reduce:animate-none text-gray-500 dark:text-gray-400" aria-hidden="true" />
		</div>
	{:else}
		<SectionPanel title="Platform limits" description="Guardrails enforced for project ownership and aggregate resource allocation.">
			<div class="grid gap-5 lg:grid-cols-3">
				<label class="block" for="user_ram_quota_gb">
					<span class="field-label">RAM quota per user</span>
					<div class="flex items-center gap-2"><input type="number" id="user_ram_quota_gb" min="0.25" max="1024" step="0.25" bind:value={settings.user_ram_quota_gb} class="field min-w-0 flex-1" aria-invalid={validationErrors.user_ram_quota_gb ? 'true' : undefined} /><span class="w-14 shrink-0 text-xs text-gray-500 dark:text-gray-400">GB</span></div>
					<p class="field-hint">Maximum aggregate project memory a user may allocate.</p>
					{#if validationErrors.user_ram_quota_gb}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.user_ram_quota_gb}</p>{/if}
				</label>
				<label class="block" for="user_cpu_quota">
					<span class="field-label">CPU quota per user</span>
					<div class="flex items-center gap-2"><input type="number" id="user_cpu_quota" min="0.1" max="256" step="0.1" bind:value={settings.user_cpu_quota} class="field min-w-0 flex-1" aria-invalid={validationErrors.user_cpu_quota ? 'true' : undefined} /><span class="w-14 shrink-0 text-xs text-gray-500 dark:text-gray-400">cores</span></div>
					<p class="field-hint">Maximum aggregate project CPU allocation for one user.</p>
					{#if validationErrors.user_cpu_quota}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.user_cpu_quota}</p>{/if}
				</label>
				<label class="block" for="max_projects">
					<span class="field-label">Maximum projects per user</span>
					<div class="flex items-center gap-2"><input type="number" id="max_projects" min="1" max="10000" step="1" bind:value={settings.max_projects} class="field min-w-0 flex-1" aria-invalid={validationErrors.max_projects ? 'true' : undefined} /><span class="w-14 shrink-0 text-xs text-gray-500 dark:text-gray-400">projects</span></div>
					<p class="field-hint">Stops new project creation after this ownership limit is reached.</p>
					{#if validationErrors.max_projects}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.max_projects}</p>{/if}
				</label>
			</div>
			<p class="mt-5 border-t border-gray-100 pt-4 text-xs text-gray-500 dark:border-neutral-800 dark:text-gray-400">These values are persisted and enforced by the running control plane after Save.</p>
		</SectionPanel>

		<SectionPanel title="Project resource defaults" description="New projects use runtime-specific profiles instead of a second global RAM/CPU default.">
			<div class="grid gap-px overflow-hidden rounded-md border border-gray-200 bg-gray-100 dark:border-neutral-800 dark:bg-neutral-800 sm:grid-cols-2 xl:grid-cols-4">
				{#each [
					{ name: 'Static', detail: '64 MB · 0.10 CPU' },
					{ name: 'Go small', detail: '128 MB · 0.20 CPU' },
					{ name: 'Node / Python', detail: '256 MB · 0.35 CPU' },
					{ name: 'Compose main', detail: '256 MB · 0.35 CPU' }
				] as profile}
					<div class="bg-white p-3 dark:bg-neutral-950">
						<p class="text-sm font-medium text-gray-950 dark:text-white">{profile.name}</p>
						<p class="metric-value mt-1 text-xs text-gray-500 dark:text-gray-400">{profile.detail}</p>
					</div>
				{/each}
			</div>
			<p class="mt-3 text-xs text-gray-500 dark:text-gray-400">The profile is selected during Create Project and can be overridden per project. Keeping profile defaults authoritative avoids a global setting silently disagreeing with the selected runtime profile.</p>
		</SectionPanel>

		<SectionPanel title="Deployment" description="Limits that affect deployment execution.">
			<div class="max-w-md">
				<label class="block" for="build_timeout_minutes">
					<span class="field-label">Build timeout</span>
					<div class="flex items-center gap-2"><input type="number" id="build_timeout_minutes" min="1" max="1440" step="1" bind:value={settings.build_timeout_minutes} class="field min-w-0 flex-1" aria-invalid={validationErrors.build_timeout_minutes ? 'true' : undefined} /><span class="w-16 shrink-0 text-xs text-gray-500 dark:text-gray-400">minutes</span></div>
					<p class="field-hint">Maximum duration for future build attempts before MyPaas times them out.</p>
					{#if validationErrors.build_timeout_minutes}<p class="mt-1 text-xs text-red-600 dark:text-red-300">{validationErrors.build_timeout_minutes}</p>{/if}
				</label>
			</div>
			<p class="mt-5 border-t border-gray-100 pt-4 text-xs text-gray-500 dark:border-neutral-800 dark:text-gray-400">Deployment concurrency remains an installation-level setting (<span class="font-mono">MAX_CONCURRENT_DEPLOYS</span>) because worker concurrency is established when the API process starts.</p>
		</SectionPanel>

		<SectionPanel title="Off-site Backup" description="Configure S3-compatible storage for automated PostgreSQL and configuration backups.">
			<div class="grid gap-5 lg:grid-cols-2">
				<label class="block">
					<span class="field-label">S3 Endpoint</span>
					<input type="text" bind:value={s3Config.endpoint} class="field" placeholder="https://s3.eu-central-1.amazonaws.com" />
				</label>
				<label class="block">
					<span class="field-label">Bucket</span>
					<input type="text" bind:value={s3Config.bucket} class="field" placeholder="mypaas-backups" />
				</label>
				<label class="block">
					<span class="field-label">Region</span>
					<input type="text" bind:value={s3Config.region} class="field" placeholder="eu-central-1" />
				</label>
				<label class="block">
					<span class="field-label">Access Key</span>
					<input type="text" bind:value={s3Config.access_key} class="field" />
				</label>
				<label class="block lg:col-span-2">
					<span class="field-label">Secret Key</span>
					<input type="password" bind:value={s3Config.secret_key} class="field" />
				</label>
			</div>
			<div class="mt-5 flex items-center gap-3 border-t border-gray-100 pt-4 dark:border-neutral-800">
				<ActionButton variant="primary" size="sm" loading={savingS3} on:click={saveS3Config}>Save S3 Config</ActionButton>
				<ActionButton variant="secondary" size="sm" loading={triggeringBackup} on:click={triggerBackup}>Trigger Manual Backup</ActionButton>
			</div>
		</SectionPanel>

		<SectionPanel title="System Update" description="Update MyPaas to the latest version and restart the control plane.">
			<div class="mt-2">
				<ActionButton variant="primary" size="sm" loading={triggeringUpdate} on:click={triggerUpdate}>Update MyPaas</ActionButton>
			</div>
		</SectionPanel>

		{#if settingsChanged}
			<div class="surface flex flex-wrap items-center justify-between gap-3 px-4 py-3">
				<div>
					<p class="inline-flex items-center gap-2 text-sm font-medium text-gray-950 dark:text-white"><span class="status-dot bg-amber-500"></span>Unsaved platform configuration</p>
					<p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">Review invalid fields before saving. Existing projects keep their own resource limits.</p>
				</div>
				<div class="flex items-center gap-2">
					<ActionButton variant="secondary" size="sm" on:click={discardChanges} disabled={savingSettings}>Discard</ActionButton>
					<ActionButton variant="primary" size="sm" loading={savingSettings} loadingLabel="Saving" on:click={saveSettings} disabled={hasValidationErrors}>Save changes</ActionButton>
				</div>
			</div>
		{/if}
	{/if}
</div>
