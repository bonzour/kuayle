import type { TimesheetFilters, TimesheetMode, TimesheetOptions, TimesheetSelection } from '$lib/types/timesheets';

export const allTimesheetFilters = (): TimesheetFilters => ({ workspaces: null, teams: null, projects: null });

export function fromSelectValues(values: string[], previous: TimesheetSelection): TimesheetSelection {
	if (values.includes('all') && previous !== null) return null;
	return values.filter((id) => id !== 'all');
}

// options may be from a bootstrap request over all currently admitted workspaces.
export function reconcileFilters(filters: TimesheetFilters, options: TimesheetOptions): TimesheetFilters {
	const workspaces = filters.workspaces === null ? null : filters.workspaces.filter((id) => options.workspaces.some((w) => w.id === id));
	const selected = new Set(workspaces ?? options.workspaces.map((w) => w.id));
	return {
		workspaces,
		teams: filters.teams === null ? null : filters.teams.filter((id) => options.teams.some((t) => t.id === id && selected.has(t.workspace_id))),
		projects: filters.projects === null ? null : filters.projects.filter((id) => id === 'none' || options.projects.some((p) => p.id === id && selected.has(p.workspace_id)))
	};
}

export function changeWorkspaces(filters: TimesheetFilters, workspaces: TimesheetSelection, options: TimesheetOptions): TimesheetFilters {
	return reconcileFilters({ ...filters, workspaces }, options);
}

export function changeMode(filters: TimesheetFilters, mode: TimesheetMode, options: TimesheetOptions): TimesheetFilters {
	if (mode !== 'personal' && options.mode !== 'personal') return filters;
	const workspaces = options.scope.workspace_ids.filter((id) =>
		options.workspaces.some((w) => w.id === id && (mode === 'personal' || w.can_administer))
	);
	return reconcileFilters({ ...filters, workspaces }, options);
}
