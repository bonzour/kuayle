import { parseDate, type CalendarDate, type DateValue } from '@internationalized/date';

// Calendar dates are not timestamps. Never interpret their UTC midnight in the browser zone.
export function parseCalendarDate(value: string | null | undefined): CalendarDate | undefined {
	if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return undefined;
	try { return parseDate(value); } catch { return undefined; }
}

export function serializeCalendarDate(value: DateValue): string {
	return value.toString();
}

// Project/cycle DATE columns are serialized by Go as midnight UTC. Use only for
// declared calendar-date fields at their adapter boundary, never for timestamps.
export function calendarDateFromApi(value: string | null | undefined): string | undefined {
	const date = value && /^\d{4}-\d{2}-\d{2}T00:00:00(?:\.0+)?Z$/.test(value) ? value.slice(0, 10) : value;
	return parseCalendarDate(date)?.toString();
}

export function formatCalendarDate(value: string, locale: string, options: Intl.DateTimeFormatOptions = {}): string {
	return parseCalendarDate(value)?.toDate('UTC').toLocaleDateString(locale, {
		day: 'numeric', month: 'short', year: 'numeric', ...options, timeZone: 'UTC'
	}) ?? '';
}
