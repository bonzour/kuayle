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
