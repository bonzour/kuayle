import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
const url = new URL('../../src/lib/features/issues/effort-worklog.ts', import.meta.url);
const source = (await readFile(url, 'utf8')).replace('./effort-duration', new URL('./effort-duration.ts', url).href);
const { validateWorklogDraft } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);

const validate = (date = '2026-10-19', duration = '1h 30m', description = '', today = '2026-10-19') => validateWorklogDraft(date, duration, description, today);
test('worklog durations require 1 through 1440 whole minutes', () => {
	for (const duration of ['1', '24h', '1440m', '1h30m']) assert.equal(validate(undefined, duration).durationValid, true);
	for (const duration of ['', '0', '-1', '24h 1m', '0.5h', '2147483648']) assert.equal(validate(undefined, duration).durationValid, false);
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
