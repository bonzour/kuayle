import type { IssueHistory } from '$lib/types/issue';

export function effortHistoryEntries(history: IssueHistory[]): IssueHistory[] {
	// Older automatic initializations of Remaining were marked "manual". A
	// companion estimate/worklog event at the same revision identifies them.
	const automatic = new Set(history.filter(entry => entry.effort && entry.effort.event_type !== 'remaining_set')
		.map(entry => `${entry.effort!.issue_id}:${entry.effort!.revision}`));
	return history.filter(entry => {
		const event = entry.effort;
		return event && (event.event_type !== 'remaining_set' ||
			event.actor.kind === 'user' && event.cause === 'manual' && !automatic.has(`${event.issue_id}:${event.revision}`));
	}).sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
		|| a.effort!.revision - b.effort!.revision || a.id.localeCompare(b.id));
}
