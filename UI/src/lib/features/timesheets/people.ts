import type { TimesheetWorklog } from '$lib/types/timesheets';

export function personLabel(person: TimesheetWorklog['owner'], unavailable: string): string {
	const name = person.available ? person.display_name || person.name || unavailable : unavailable;
	return `${name} · ${person.id.slice(-8)}`;
}
