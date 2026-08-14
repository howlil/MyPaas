import { describe, expect, it } from 'vitest';

import { projectCreationReadiness } from './project';

const readyDockerfile = {
	name: 'demo-app',
	sourceType: 'git' as const,
	sourceReady: true,
	deployMode: 'dockerfile',
	appPort: '3000',
	composeDisabledReason: '',
	busy: false
};

describe('Create Project runtime readiness contract', () => {
	it('fails closed when repository inspection is stale or incomplete', () => {
		const result = projectCreationReadiness({
			...readyDockerfile,
			sourceReady: false
		});
		expect(result.ready).toBe(false);
		expect(result.reason).toMatch(/repository|source/i);
	});

	it('fails closed while any repository or runtime analysis work is busy', () => {
		const result = projectCreationReadiness({
			...readyDockerfile,
			busy: true
		});
		expect(result).toEqual({
			ready: false,
			state: 'Analyzing deployment',
			reason: 'Runtime analysis must finish before this project can be created'
		});
	});

	it('does not let a compose blocker become ready even with service and port resolved', () => {
		const result = projectCreationReadiness({
			...readyDockerfile,
			deployMode: 'compose',
			mainService: 'api',
			composeDisabledReason: 'Compose Doctor found an invalid public service'
		});
		expect(result.ready).toBe(false);
		expect(result.reason).toContain('Compose Doctor');
	});

	it('does not let missing required compose environment values become ready', () => {
		const result = projectCreationReadiness({
			...readyDockerfile,
			deployMode: 'compose',
			mainService: 'api',
			composeDisabledReason: 'Fill required env values: DATABASE_URL'
		});
		expect(result.ready).toBe(false);
		expect(result.reason).toContain('DATABASE_URL');
	});

	it('keeps registry images blocked until a port is provided', () => {
		const result = projectCreationReadiness({
			name: 'registry-app',
			sourceType: 'registry',
			sourceReady: true,
			deployMode: 'image',
			appPort: '',
			composeDisabledReason: '',
			busy: false
		});
		expect(result.ready).toBe(false);
		expect(result.reason).toMatch(/container port is required/i);
	});

	it('allows a resolved static contract without inventing a runtime port', () => {
		const result = projectCreationReadiness({
			name: 'static-app',
			sourceType: 'git',
			sourceReady: true,
			deployMode: 'static',
			appPort: '',
			composeDisabledReason: '',
			busy: false
		});
		expect(result.ready).toBe(true);
	});
});
