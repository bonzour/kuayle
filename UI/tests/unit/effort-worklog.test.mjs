import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
const url = new URL('../../src/lib/features/issues/effort-worklog.ts', import.meta.url);
const source = (await readFile(url, 'utf8')).replace('./effort-duration', new URL('./effort-duration.ts', url).href);
const { validateWorklogDraft, worklogPeriodDays } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);

const validate = (date = '2026-10-19', duration = '1h 30m', description = '', today = '2026-10-19') => validateWorklogDraft(date, duration, description, today);
test('worklog durations require 1 through 1440 whole minutes', () => {
	for (const duration of ['1m', '1', '0.5d', '3d', '24h', '1440m', '1h30m']) assert.equal(validate(undefined, duration).durationValid, true);
	for (const duration of ['', '0', '-1', '24h 1m', '3d 1m', '0.5h', '2147483648']) assert.equal(validate(undefined, duration).durationValid, false);
});
test('calendar dates are checked without browser timezone conversions', () => {
	for (const date of ['0001-01-01', '2000-02-29', '2024-02-29', '2026-10-19']) assert.equal(validate(date).dateValid, true, date);
	for (const date of ['', '0000-01-01', '1900-02-29', '2026-02-29', '2026-04-31', '2026-13-01', '2026-10-20', '2026-1-01']) assert.equal(validate(date).dateValid, false, date);
	assert.equal(validate('2026-10-19', '60', '', '2026-10-18').dateValid, false, 'workspace today controls future dates');
});
test('description counts Unicode code points without truncating multiline text', () => {
	const text = '🙂'.repeat(1999) + '\n';
	assert.equal(validate(undefined, undefined, text).descriptionValid, true);
	assert.equal(validate(undefined, undefined, text + '界').descriptionValid, false);
	assert.equal(Array.from(text).length, 2000);
});

test('periods include both endpoints and skip weekends using calendar dates', () => {
	assert.equal(worklogPeriodDays('2026-01-02', '2026-01-05', true), 2);
	assert.equal(worklogPeriodDays('2026-01-02', '2026-01-05', false), 4);
	assert.equal(worklogPeriodDays('2026-03-27', '2026-03-30', true), 2);
	assert.equal(worklogPeriodDays('2024-02-28', '2024-03-01', true), 3);
	assert.equal(worklogPeriodDays('0001-01-01', '0001-01-01', false), 1);
	assert.equal(worklogPeriodDays('2026-01-01', '2026-01-14', true), 10);
	assert.equal(worklogPeriodDays('2026-01-04', '2026-01-11', true), 5);
	for (const [from, to] of [['2026-01-03', '2026-01-04'], ['2026-01-05', '2026-01-02'], ['2026-02-30', '2026-03-01']]) assert.equal(worklogPeriodDays(from, to, true), 0);
});

const historySource = await readFile(new URL('../../src/lib/features/issues/effort-history.ts', import.meta.url), 'utf8');
const { effortHistoryEntries } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(historySource)).toString('base64')}`);
test('work log history includes only manual Remaining, including legacy automatic initializations', () => {
	const entry = (id, event_type, revision, cause = 'manual', kind = 'user', issue_id = 'issue') => ({ id, created_at: id, effort: { issue_id, event_type, revision, cause, actor: { kind } } });
	const rows = [entry('1', 'estimate_set', 1), entry('2', 'remaining_set', 1), entry('3', 'worklog_created', 2), entry('4', 'remaining_set', 2, 'worklog'), entry('5', 'remaining_set', 3, 'closure', 'system'), entry('6', 'remaining_set', 4), entry('7', 'remaining_set', 5, 'reactivation'), { id: '8', field: 'title' }, entry('9', 'remaining_set', 1, 'manual', 'user', 'other')];
	assert.deepEqual(effortHistoryEntries(rows).map(row => row.id), ['1', '3', '6', '9']);
});

test('the latest work log event uses revision order when timestamps match', () => {
	const entry = (id, revision) => ({ id, created_at: '2026-10-20T12:00:00Z', effort: { issue_id: 'issue', event_type: 'worklog_created', revision } });
	const entries = effortHistoryEntries([entry('a', 3), entry('z', 1), entry('m', 2)]);
	assert.deepEqual(entries.map(row => row.id), ['z', 'm', 'a']);
	assert.equal(entries.at(-1).effort.revision, 3);
});
