import assert from 'node:assert/strict';
import test from 'node:test';
import { parseCalendarDate, serializeCalendarDate, formatCalendarDate, calendarDateFromApi } from '../../src/lib/utils/calendar-date.ts';

test('calendar dates round trip and format identically in all supported reference zones', () => {
	const previous = process.env.TZ;
	try {
		for (const zone of ['Europe/Rome', 'America/Los_Angeles', 'Pacific/Kiritimati']) {
			process.env.TZ = zone;
			for (const value of ['2026-10-21', '2024-02-29', '2025-12-31', '2026-01-01', '2026-03-29', '0001-01-01']) {
				assert.equal(serializeCalendarDate(parseCalendarDate(value)), value);
			}
			assert.equal(formatCalendarDate('2026-10-21', 'it', { month: '2-digit' }), '21/10/2026');
		}
	} finally { if (previous === undefined) delete process.env.TZ; else process.env.TZ = previous; }
});

test('missing, malformed and impossible dates are never coerced into another date', () => {
	for (const value of [null, undefined, '', '2026-02-29', '2024-02-30', '2026-13-01', '2026-10-21T00:00:00Z', '21/10/2026']) {
		assert.equal(parseCalendarDate(value), undefined);
		assert.equal(formatCalendarDate(value ?? '', 'en'), '');
	}
});

test('calendar API adapter accepts DATE-column serialization without converting real timestamps', () => {
	assert.equal(calendarDateFromApi('2026-10-21T00:00:00Z'), '2026-10-21');
	assert.equal(calendarDateFromApi('2026-10-21'), '2026-10-21');
	assert.equal(calendarDateFromApi('2026-10-21T12:00:00Z'), undefined);
	assert.equal(calendarDateFromApi('2026-02-30T00:00:00Z'), undefined);
});
