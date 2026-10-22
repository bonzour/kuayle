import test from 'node:test';
import assert from 'node:assert/strict';
import { presentationKey, retainedGroups } from '../../src/lib/features/timesheets/presentation.ts';

test('presentation is retained only for the same identity, report and current capabilities', () => {
	const query = { mode: 'personal', from: '2026-10-19', to: '2026-10-25', filters: { workspaces: null, teams: null, projects: null }, owners: null, page: 1 };
	const grid = { scope: { workspace_ids: ['a'] }, reference: { id: 'a' }, workspaces: [{ id: 'a', can_record_work: true, can_administer: false }] };
	const baseline = presentationKey('user', query, grid);
	assert.equal(baseline, presentationKey('user', structuredClone(query), { ...grid, total_minutes: 800, groups: [] }));
	for (const changed of [{ ...query, page: 2 }, { ...query, owners: [] }, { ...query, mode: 'person' }, { ...query, from: '2026-10-12' }]) assert.notEqual(baseline, presentationKey('user', changed, grid));
	assert.notEqual(baseline, presentationKey('other', query, grid));
	assert.notEqual(baseline, presentationKey('user', query, { ...grid, workspaces: [{ ...grid.workspaces[0], can_record_work: false }] }));
	assert.notEqual(baseline, presentationKey('user', query, { ...grid, scope: { workspace_ids: [] } }));
});

test('removed groups are discarded without caching any report records', () => {
	assert.deepEqual(retainedGroups(['parent', 'child', 'revoked'], [{ key: 'parent', children: [{ key: 'child', children: [] }] }]), ['parent', 'child']);
});
