import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';

const source = await readFile(new URL('../../src/lib/features/timesheets/selection.ts', import.meta.url), 'utf8');
const { allTimesheetFilters, fromSelectValues, reconcileFilters, changeWorkspaces, changeMode } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);
const options = {
	mode: 'personal', available_modes: ['personal', 'person', 'project', 'workspace', 'team'],
	workspaces: [{ id: 'a', can_administer: true }, { id: 'b', can_administer: false }],
	teams: [{ id: 'ta', workspace_id: 'a' }, { id: 'tb', workspace_id: 'b' }],
	projects: [{ id: 'pa', workspace_id: 'a' }, { id: 'pb', workspace_id: 'b' }],
	scope: { workspace_ids: ['a', 'b'] }
};

test('All, an empty selection, and no project retain distinct meanings', () => {
	assert.deepEqual(allTimesheetFilters(), { workspaces: null, teams: null, projects: null });
	assert.deepEqual(fromSelectValues([], null), []);
	assert.deepEqual(fromSelectValues(['all', 'a'], null), ['a']);
	assert.equal(fromSelectValues(['a', 'all'], ['a']), null);
	assert.deepEqual(fromSelectValues(['none'], null), ['none']);
});

test('removing a workspace only removes incompatible child filters', () => {
	const filters = { workspaces: null, teams: ['ta', 'tb'], projects: ['pa', 'pb', 'none'] };
	assert.deepEqual(changeWorkspaces(filters, ['b'], options), {
		workspaces: ['b'], teams: ['tb'], projects: ['pb', 'none']
	});
	assert.deepEqual(changeWorkspaces(filters, [], options), { workspaces: [], teams: [], projects: ['none'] });
});

test('an administrative switch intersects effective workspaces, preserving empty selections', () => {
	assert.deepEqual(changeMode(allTimesheetFilters(), 'person', options), { workspaces: ['a'], teams: null, projects: null });
	const memberOnly = { ...options, scope: { workspace_ids: ['b'] } };
	assert.deepEqual(changeMode({ workspaces: ['b'], teams: ['tb'], projects: ['pb'] }, 'team', memberOnly), {
		workspaces: [], teams: [], projects: []
	});
	const empty = { ...options, scope: { workspace_ids: [] } };
	assert.deepEqual(changeMode({ workspaces: [], teams: [], projects: [] }, 'person', empty), { workspaces: [], teams: [], projects: [] });
});

test('administrative group switches preserve filters; returning to personal does not broaden scope', () => {
	const administrative = { ...options, mode: 'person', workspaces: [options.workspaces[0]], scope: { workspace_ids: ['a'] } };
	const filters = { workspaces: null, teams: ['ta'], projects: ['none'] };
	assert.deepEqual(changeMode(filters, 'project', administrative), filters);
	assert.deepEqual(changeMode(filters, 'personal', administrative), { ...filters, workspaces: ['a'] });
});

test('revoked explicit scopes become empty, never All', () => {
	const admitted = { ...options, workspaces: [], teams: [], projects: [] };
	assert.deepEqual(reconcileFilters({ workspaces: ['a'], teams: ['ta'], projects: ['pa'] }, admitted), {
		workspaces: [], teams: [], projects: []
	});
	assert.deepEqual(reconcileFilters(allTimesheetFilters(), admitted), allTimesheetFilters());
});

test('the API serializes explicit empty filters instead of omitting them', async () => {
	const source = (await readFile(new URL('../../src/lib/api/timesheets.ts', import.meta.url), 'utf8'))
		.replace("import { api } from './client';", 'const api = { get: async (url) => url };');
	const { getTimesheetOptions } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);
	const url = new URL(await getTimesheetOptions('personal', { workspaces: [], teams: null, projects: ['pa', 'none'] }), 'http://test');
	assert.equal(url.pathname, '/api/timesheets/options');
	assert.equal(url.searchParams.get('workspace_ids'), '');
	assert.equal(url.searchParams.get('team_ids'), 'all');
	assert.equal(url.searchParams.get('project_ids'), 'pa,none');
});
