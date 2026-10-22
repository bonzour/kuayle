import type { TimesheetGrid, TimesheetQuery } from '$lib/types/timesheets';

export interface TimesheetPresentation {
	collapsed: string[];
	scrollTop: number;
	scrollLeft: number;
	pageScroll: number;
	height: number;
	focus: string | null;
}

// This key contains identity/capabilities, never labels or worklog contents.
export function presentationKey(user: string | undefined, query: TimesheetQuery, grid: TimesheetGrid): string {
	return JSON.stringify([user, query, grid.scope, grid.reference.id,
		grid.workspaces.map(w => [w.id, w.can_record_work, w.can_administer])]);
}

export function retainedGroups(collapsed: string[], groups: TimesheetGrid['groups']): string[] {
	const keys = new Set<string>();
	function visit(nodes: typeof groups) { for (const node of nodes) { keys.add(node.key); visit(node.children); } }
	visit(groups);
	return collapsed.filter(key => keys.has(key));
}
