import { CalendarDate, parseDate } from '@internationalized/date';
import { MINUTES_PER_EFFORT_DAY } from '../issues/effort-duration.ts';
import { formatCalendarDate } from '../../utils/calendar-date.ts';
import type { TimesheetPeriod } from '$lib/types/timesheets';

export function periodForDate(period: TimesheetPeriod['period'], value: string): TimesheetPeriod {
	const date = parseDate(value);
	const from = period === 'month'
		? new CalendarDate(date.year, date.month, 1)
		: date.subtract({ days: (date.toDate('UTC').getUTCDay() + 6) % 7 });
	const to = period === 'month' ? new CalendarDate(from.year, from.month, from.calendar.getDaysInMonth(from)) : from.add({ days: 6 });
	return { period, from: from.toString(), to: to.toString() };
}

export function movePeriod(value: TimesheetPeriod, direction: -1 | 1): TimesheetPeriod {
	const anchor = parseDate(value.from).add(value.period === 'week' ? { days: direction * 7 } : { months: direction });
	return periodForDate(value.period, anchor.toString());
}

export function formatTimesheetMinutes(minutes: number): string {
	const hours = Math.floor(minutes / 60);
	const rest = minutes % 60;
	return [hours ? `${hours}h` : '', rest || !hours ? `${rest}m` : ''].filter(Boolean).join(' ');
}

export function formatTimesheetHours(minutes: number, locale: string): string {
	return `${new Intl.NumberFormat(locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(minutes / 60)}h`;
}

export function formatTimesheetDays(minutes: number, locale: string): string {
	return `${new Intl.NumberFormat(locale, { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(minutes / MINUTES_PER_EFFORT_DAY)}d`;
}

export function formatTimesheetDate(value: string, locale: string, options: Intl.DateTimeFormatOptions = {}): string {
	return formatCalendarDate(value, locale, { year: undefined, ...options });
}
