import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';

const source = (await readFile(new URL('../../src/lib/features/timesheets/period.ts', import.meta.url), 'utf8'))
	.replace("'../issues/effort-duration.ts'", JSON.stringify(new URL('../../src/lib/features/issues/effort-duration.ts', import.meta.url).href))
	.replace("'../../utils/calendar-date.ts'", JSON.stringify(new URL('../../src/lib/utils/calendar-date.ts', import.meta.url).href))
	.replace("'@internationalized/date'", JSON.stringify(import.meta.resolve('@internationalized/date')));
const { periodForDate, movePeriod, formatTimesheetMinutes, formatTimesheetHours, formatTimesheetDays, formatTimesheetDate } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);

test('calendar periods include Monday through Sunday, leap years and the entire month', () => {
	assert.deepEqual(periodForDate('week', '2026-01-01'), { period: 'week', from: '2025-12-29', to: '2026-01-04' });
	assert.deepEqual(periodForDate('week', '2026-10-25'), { period: 'week', from: '2026-10-19', to: '2026-10-25' });
	assert.deepEqual(periodForDate('month', '2024-02-15'), { period: 'month', from: '2024-02-01', to: '2024-02-29' });
	assert.equal(periodForDate('month', '2023-02-15').to, '2023-02-28');
	assert.equal(periodForDate('month', '9999-12-01').to, '9999-12-31');
	assert.throws(() => periodForDate('week', '2026-02-30'));
});

test('navigation crosses month/year boundaries and switching uses the current start date', () => {
	const week = periodForDate('week', '2026-01-01');
	assert.deepEqual(movePeriod(week, 1), { period: 'week', from: '2026-01-05', to: '2026-01-11' });
	assert.deepEqual(movePeriod(movePeriod(week, -1), 1), week);
	const month = periodForDate('month', week.from);
	assert.equal(month.from, '2025-12-01');
	assert.equal(movePeriod(month, 1).from, '2026-01-01');
});

test('browser timezone and DST never shift calendar dates or reference today', () => {
	const previous = process.env.TZ;
	try {
		for (const tz of ['Europe/Rome', 'America/Adak', 'Pacific/Kiritimati', 'UTC']) {
			process.env.TZ = tz;
			assert.deepEqual(periodForDate('week', '2026-03-29'), { period: 'week', from: '2026-03-23', to: '2026-03-29' });
			assert.equal(periodForDate('week', '2026-10-25').to, '2026-10-25');
			assert.equal(formatTimesheetDate('2026-03-29', 'en-GB', { month: '2-digit', day: '2-digit' }), '29/03');
			assert.equal(periodForDate('week', '2026-10-25').from, '2026-10-19');
			assert.equal(periodForDate('week', '2026-10-26').from, '2026-10-26');
		}
	} finally { if (previous === undefined) delete process.env.TZ; else process.env.TZ = previous; }
});

test('durations stay exact hours and minutes, including totals greater than 24h', () => {
	for (const [minutes, expected] of [[0,'0m'],[1,'1m'],[60,'1h'],[90,'1h 30m'],[480,'8h'],[1500,'25h'],[2400,'40h']]) assert.equal(formatTimesheetMinutes(minutes), expected);
});

test('grid hours use two fixed localized decimals, including rounding and whole hours', () => {
	for (const [minutes, expected] of [[0, '0.00h'], [1, '0.02h'], [5, '0.08h'], [30, '0.50h'], [51, '0.85h'], [60, '1.00h'], [90, '1.50h'], [111, '1.85h'], [480, '8.00h'], [1500, '25.00h']]) {
		assert.equal(formatTimesheetHours(minutes, 'en'), expected);
		assert.equal(formatTimesheetHours(minutes, 'it'), expected.replace('.', ','));
	}
});

test('totals always use eight-hour days rounded to two fixed localized decimals', () => {
	for (const [minutes, expected] of [[0, '0.00d'], [1, '0.00d'], [30, '0.06d'], [90, '0.19d'], [240, '0.50d'], [480, '1.00d'], [510, '1.06d'], [1500, '3.13d'], [2400, '5.00d']]) {
		assert.equal(formatTimesheetDays(minutes, 'en'), expected);
		assert.equal(formatTimesheetDays(minutes, 'it'), expected.replace('.', ','));
	}
});

test('grid and detail serialize the same period/scope and keep explicit empty filters', async () => {
	const source = (await readFile(new URL('../../src/lib/api/timesheets.ts', import.meta.url), 'utf8'))
		.replace("import { api } from './client';", 'const api = { get: async (url) => url };');
	const { getTimesheet, getTimesheetWorklogs } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);
	const input = { mode: 'personal', owners: null, ...periodForDate('week', '2026-01-01'), reference_workspace_id: 'reference', page: 2, filters: { workspaces: [], teams: null, projects: ['none'] } };
	const grid = new URL(await getTimesheet(input), 'http://test');
	const detail = new URL(await getTimesheetWorklogs(input, 'issue', '2025-12-31'), 'http://test');
	assert.equal(grid.pathname, '/api/timesheets');
	assert.equal(detail.pathname, '/api/timesheets/worklogs');
	for (const [key,value] of grid.searchParams) assert.equal(detail.searchParams.get(key), value);
	assert.equal(grid.searchParams.get('workspace_ids'), '');
	assert.equal(grid.searchParams.get('page'), '2');
	assert.equal(detail.searchParams.get('issue_id'), 'issue');
});

test('administrative modes serialize identical filters into grid and detail, including empty people', async () => {
 const source = (await readFile(new URL('../../src/lib/api/timesheets.ts', import.meta.url), 'utf8')).replace("import { api } from './client';", 'const api = { get: async (url) => url };');
 const { getTimesheet, getTimesheetWorklogs } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);
 for (const mode of ['person','project','workspace','team']) {
  for (const owners of [null, [], ['one','two']]) {
   const input = { mode, owners, period:'week',from:'2026-01-05',to:'2026-01-11',reference_workspace_id:'reference',page:1,filters:{workspaces:null,teams:null,projects:['none']} };
   const grid = new URL(await getTimesheet(input),'http://test');
   const detail = new URL(await getTimesheetWorklogs(input,'issue','2026-01-05'),'http://test');
   assert.equal(grid.searchParams.get('mode'),mode);
   assert.equal(grid.searchParams.get('owner_user_ids'), owners === null ? 'all' : owners.join(','));
   for (const [key,value] of grid.searchParams) assert.equal(detail.searchParams.get(key),value);
  }
 }
});

test('historical person labels remain distinct and never invent unavailable names', async () => {
 const source = await readFile(new URL('../../src/lib/features/timesheets/people.ts', import.meta.url),'utf8');
 const {personLabel} = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);
 assert.equal(personLabel({id:'account-12345678',available:false,name:'Old name',display_name:'Old display'},'Unavailable'),'Unavailable · 12345678');
 assert.equal(personLabel({id:'account-87654321',available:true,name:'Same',display_name:null},'Unavailable'),'Same · 87654321');
});
