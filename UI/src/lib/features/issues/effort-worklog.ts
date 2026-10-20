import { parseEffortMinutes } from './effort-duration';

export function validateWorklogDraft(date: string, duration: string, description: string, today: string) {
	const minutes = parseEffortMinutes(duration);
	const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
	let validDate = false;
	if (match) {
		const [, y, mo, d] = match.map(Number);
		const leap = y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0);
		const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
		validDate = y >= 1 && mo >= 1 && mo <= 12 && d >= 1 && d <= days[mo - 1] && date <= today;
	}
	return {
		minutes,
		dateValid: validDate,
		durationValid: minutes !== null && minutes >= 1 && minutes <= 1440,
		descriptionValid: Array.from(description).length <= 2000
	};
}

export function worklogPeriodDays(start: string, end: string, skipWeekends: boolean): number {
	if (!validateWorklogDraft(start, '1m', '', '9999-12-31').dateValid ||
		!validateWorklogDraft(end, '1m', '', '9999-12-31').dateValid || end < start) return 0;
	const date = new Date(`${start}T00:00:00Z`);
	const days = (new Date(`${end}T00:00:00Z`).getTime() - date.getTime()) / 86400000 + 1;
	if (!skipWeekends) return days;
	let selected = Math.floor(days / 7) * 5;
	for (let i = 0; i < days % 7; i++) {
		const weekday = (date.getUTCDay() + i) % 7;
		if (weekday !== 0 && weekday !== 6) selected++;
	}
	return selected;
}
