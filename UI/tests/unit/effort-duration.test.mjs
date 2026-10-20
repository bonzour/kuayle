import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';

const source = await readFile(new URL('../../src/lib/features/issues/effort-duration.ts', import.meta.url), 'utf8');
const { parseEffortMinutes, formatEffortMinutes } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);

test('whole minutes, hours and combinations preserve minute precision', () => {
	for (const [text, expected] of [['0', 0], ['90m', 90], ['2h', 120], ['1h 30m', 90], ['1h30m', 90], [' 2H 5M ', 125], ['2147483647m', 2147483647], ['1', 480], ['0.5', 240], ['1d', 480], ['0,5d', 240], ['1.025d', 492], ['1d 2h 3m', 603], ['0.1d', 48]]) {
		assert.equal(parseEffortMinutes(text), expected, text);
		assert.equal(parseEffortMinutes(formatEffortMinutes(expected)), expected);
	}
});

test('invalid or overflowing durations are rejected without rounding', () => {
	for (const text of ['', ' ', '-1', '+1', '0.001d', '1.5h', '0.1m', '1,5h', '1d 2d', '1w', '1m 1h', '1h 2h', '1h garbage', '2147483648', '35791395h', '9999999999999999999999999999999999999']) {
		assert.equal(parseEffortMinutes(text), null, text);
	}
});

test('days are the primary display unit and always round-trip exactly', () => {
	for (const [minutes, text] of [[0, '0d'], [480, '1d'], [720, '1d 4h'], [1501, '3d 1h 1m'], [1, '1m']]) assert.equal(formatEffortMinutes(minutes), text);
});
