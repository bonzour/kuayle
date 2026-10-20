import { api } from './client';
import type { TimesheetFilters, TimesheetMode, TimesheetOptions, TimesheetQuery, TimesheetGrid, TimesheetWorklogs } from '$lib/types/timesheets';

export function getTimesheetOptions(mode: TimesheetMode, filters: TimesheetFilters): Promise<TimesheetOptions> {
	return api.get(`/api/timesheets/options?${timesheetFiltersQuery(mode, filters)}`);
}

function timesheetFiltersQuery(mode: TimesheetMode, filters: TimesheetFilters): URLSearchParams {
	const query = new URLSearchParams({ mode });
	for (const [key, selection] of Object.entries(filters)) {
		const field = key === 'workspaces' ? 'workspace_ids' : key === 'teams' ? 'team_ids' : 'project_ids';
		query.set(field, selection === null ? 'all' : selection.join(','));
	}
	return query;
}

function timesheetReadQuery(input: TimesheetQuery): URLSearchParams {
	const query = timesheetFiltersQuery(input.mode, input.filters);
	if (input.mode !== 'personal') query.set('owner_user_ids', input.owners === null ? 'all' : input.owners.join(','));
	for (const key of ['period', 'from', 'to', 'reference_workspace_id', 'page'] as const) query.set(key, String(input[key]));
	return query;
}

export function getTimesheet(input: TimesheetQuery): Promise<TimesheetGrid> {
	return api.get(`/api/timesheets?${timesheetReadQuery(input)}`);
}

export function getTimesheetWorklogs(input: TimesheetQuery, issueId: string, workDate: string): Promise<TimesheetWorklogs> {
	const query = timesheetReadQuery(input);
	query.set('issue_id', issueId);
	query.set('work_date', workDate);
	return api.get(`/api/timesheets/worklogs?${query}`);
}
