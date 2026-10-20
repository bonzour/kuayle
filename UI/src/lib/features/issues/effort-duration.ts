export const MAX_EFFORT_MINUTES = 2147483647;
export const MINUTES_PER_EFFORT_DAY = 480;

export function parseEffortMinutes(input: string): number | null {
	const match = /^(?:(\d+(?:[.,]\d+)?)\s*d(?:\s*(\d+)\s*h)?(?:\s*(\d+)\s*m)?|(\d+)\s*h(?:\s*(\d+)\s*m)?|(\d+)\s*m|(\d+(?:[.,]\d+)?))$/i.exec(input.trim());
	if (!match) return null;
	const [whole, fraction = ''] = (match[1] ?? match[7] ?? '0').replace(',', '.').split('.');
	const denominator = 10n ** BigInt(fraction.length);
	const dayMinutes = BigInt(whole + fraction) * BigInt(MINUTES_PER_EFFORT_DAY);
	if (dayMinutes % denominator !== 0n) return null;
	const minutes = dayMinutes / denominator + BigInt(match[2] ?? match[4] ?? '0') * 60n + BigInt(match[3] ?? match[5] ?? match[6] ?? '0');
	return minutes <= BigInt(MAX_EFFORT_MINUTES) ? Number(minutes) : null;
}

export function formatEffortMinutes(minutes: number): string {
	if (minutes === 0) return '0d';
	const days = Math.floor(minutes / MINUTES_PER_EFFORT_DAY);
	const hours = Math.floor(minutes % MINUTES_PER_EFFORT_DAY / 60);
	const rest = minutes % 60;
	return [days ? `${days}d` : '', hours ? `${hours}h` : '', rest ? `${rest}m` : ''].filter(Boolean).join(' ');
}
