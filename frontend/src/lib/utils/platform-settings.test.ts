import { describe, expect, it } from 'vitest';

const editablePlatformSettings = [
	'user_ram_quota_gb',
	'user_cpu_quota',
	'max_projects',
	'build_timeout_minutes'
] as const;

describe('admin platform settings contract', () => {
	it('does not expose dead project-default or startup-only controls', () => {
		expect(editablePlatformSettings).not.toContain('project_default_ram_mb');
		expect(editablePlatformSettings).not.toContain('project_default_cpu');
		expect(editablePlatformSettings).not.toContain('max_concurrent_deploys');
	});
});
