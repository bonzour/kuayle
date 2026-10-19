export const MAX_EFFORT_MINUTES = 2147483647;

export function parseEffortMinutes(input: string): number | null {
	const match = /^(?:(\d+)\s*h(?:\s*(\d+)\s*m)?|(\d+)\s*m|(\d+))$/i.exec(input.trim());
	if (!match) return null;
	const minutes = match[1] !== undefined
		? Number(match[1]) * 60 + Number(match[2] ?? 0)
		: Number(match[3] ?? match[4]);
	return Number.isSafeInteger(minutes) && minutes <= MAX_EFFORT_MINUTES ? minutes : null;
}

export function formatEffortMinutes(minutes: number): string {
	const hours = Math.floor(minutes / 60);
	const rest = minutes % 60;
	return hours ? `${hours}h${rest ? ` ${rest}m` : ''}` : `${rest}m`;
}
