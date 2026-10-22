import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';

const moduleUrl = source => `data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`;
const periodSource = (await readFile(new URL('../../src/lib/features/timesheets/period.ts', import.meta.url), 'utf8'))
	.replace("'../issues/effort-duration.ts'", JSON.stringify(new URL('../../src/lib/features/issues/effort-duration.ts', import.meta.url).href))
	.replace("'../../utils/calendar-date.ts'", JSON.stringify(new URL('../../src/lib/utils/calendar-date.ts', import.meta.url).href))
	.replace("'@internationalized/date'", JSON.stringify(import.meta.resolve('@internationalized/date')));
const source = (await readFile(new URL('../../src/lib/features/timesheets/navigation.ts', import.meta.url), 'utf8'))
	.replace("'./period'", JSON.stringify(moduleUrl(periodSource)));
const { readTimesheetLocation, writeTimesheetLocation } = await import(moduleUrl(source));
const read = query => readTimesheetLocation(new URLSearchParams(query));
const referenceId = '00000000-0000-0000-0000-000000000001';

test('bare and partial URLs defer current dates to the server reference', () => {
	assert.deepEqual(read(''), {
		mode: 'personal', owners: null, filters: { workspaces: null, teams: null, projects: null },
		period: null, periodKind: 'week', referenceId: '', groupPage: 1
	});
	assert.equal(read(`period=month&reference_workspace_id=${referenceId}`).periodKind, 'month');
	assert.equal(read(`period=month&reference_workspace_id=${referenceId}`).period, null);
});

test('all five modes round trip dates, reference, filters, people and group page', () => {
	for (const mode of ['personal', 'person', 'project', 'workspace', 'team']) {
		for (const ids of [null, [], ['one', 'two']]) {
			const value = {
				mode, owners: mode === 'personal' ? null : ids,
				filters: { workspaces: ids, teams: ids, projects: ids === null ? null : [...ids, 'none'] },
				period: { period: 'month', from: '2024-02-01', to: '2024-02-29' }, periodKind: 'month',
				referenceId, groupPage: 2
			};
			const url = writeTimesheetLocation(new URL('https://app.test/a/timesheet?unrelated=keep#grid'), value);
			assert.deepEqual(readTimesheetLocation(url.searchParams), value);
			assert.equal(url.pathname, '/a/timesheet');
			assert.equal(url.hash, '#grid');
			assert.equal(url.searchParams.get('unrelated'), 'keep');
			assert.equal(url.searchParams.get('workspace_ids'), ids === null ? 'all' : ids.join(','));
			assert.equal(url.searchParams.has('owner_user_ids'), mode !== 'personal');
			assert.equal(writeTimesheetLocation(url, value).href, url.href);
		}
	}
});

test('personal URLs never restore somebody else as owner; lists deduplicate without broadening', () => {
	assert.equal(read('mode=personal&owner_user_ids=another').owners, null);
	assert.deepEqual(read('workspace_ids=a,a,b&team_ids=&project_ids=none,none').filters, {
		workspaces: ['a', 'b'], teams: [], projects: ['none']
	});
});

test('malformed URLs fail instead of resetting a filter to All', () => {
	for (const query of [
		'mode=unknown', 'mode=', 'mode=personal&mode=person', 'period=day', 'period=',
		'from=2026-10-19', 'to=2026-10-25', 'from=2026-10-20&to=2026-10-26',
		'from=2026-10-19&to=2026-10-24', 'from=0000-01-03&to=0000-01-09',
		'period=month&from=2023-02-01&to=2023-02-29', 'period=month&from=2024-02-02&to=2024-02-29',
		'page=0', 'page=-1', 'page=1.5', 'page=Infinity', 'page=9007199254740991', 'page=1&page=2',
		'workspace_ids=all,a', 'team_ids=none', 'workspace_ids=a,', 'project_ids=none,,a',
		'project_ids=%20a', 'reference_workspace_id=', 'reference_workspace_id=unknown',
		'reference_workspace_id=00000000-0000-0000-0000-000000000000', 'mode=person&owner_user_ids=a,'
	]) assert.throws(() => read(query), undefined, query);
});

test('historic dates across years and DST survive URL round trips in every browser timezone', () => {
	const previous = process.env.TZ;
	try {
		for (const timezone of ['Europe/Rome', 'Pacific/Kiritimati', 'America/Adak']) {
			process.env.TZ = timezone;
			for (const dates of ['from=2025-12-29&to=2026-01-04', 'from=2026-03-23&to=2026-03-29', 'from=2026-10-19&to=2026-10-25']) {
				const value = read(dates);
				assert.deepEqual(readTimesheetLocation(writeTimesheetLocation(new URL('https://app.test/a/timesheet'), value).searchParams), value);
			}
		}
	} finally { if (previous === undefined) delete process.env.TZ; else process.env.TZ = previous; }
});
