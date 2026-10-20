import type { TimesheetFilters, TimesheetMode, TimesheetPeriod, TimesheetSelection } from '$lib/types/timesheets';
import { periodForDate } from './period';

export interface TimesheetLocation {
	mode: TimesheetMode;
	owners: TimesheetSelection;
	filters: TimesheetFilters;
	period: TimesheetPeriod | null;
	periodKind: TimesheetPeriod['period'];
	referenceId: string;
	groupPage: number;
}

const modes: TimesheetMode[] = ['personal', 'person', 'project', 'workspace', 'team'];
const fields = ['mode', 'period', 'from', 'to', 'reference_workspace_id', 'workspace_ids', 'team_ids', 'project_ids', 'owner_user_ids', 'page'];

export function readTimesheetLocation(params: URLSearchParams): TimesheetLocation {
	const invalid = () => { throw new Error('Invalid Timesheet URL'); };
	for (const key of fields) if (params.getAll(key).length > 1) invalid();
	const mode = (params.get('mode') ?? 'personal') as TimesheetMode;
	const periodKind = params.get('period') ?? 'week';
	if (!modes.includes(mode) || (periodKind !== 'week' && periodKind !== 'month')) invalid();
	const selection = (key: string): TimesheetSelection => {
		const raw = params.get(key);
		if (raw === null || raw === 'all') return null;
		if (raw === '') return [];
		const ids = raw.split(',');
		if (ids.some(id => !id || id.trim() !== id || id === 'all' || (id === 'none' && key !== 'project_ids'))) invalid();
		return [...new Set(ids)];
	};
	let period: TimesheetPeriod | null = null;
	if (params.has('from') || params.has('to')) {
		const from = params.get('from') ?? '';
		const to = params.get('to') ?? '';
		if (![from, to].every(date => /^(?!0000)\d{4}-\d{2}-\d{2}$/.test(date))) invalid();
		period = periodForDate(periodKind as TimesheetPeriod['period'], from);
		if (period.from !== from || period.to !== to) invalid();
	}
	const rawPage = params.get('page') ?? '1';
	const groupPage = Number(rawPage);
	if (!/^[1-9]\d*$/.test(rawPage) || !Number.isSafeInteger(groupPage) || groupPage > Math.floor(Number.MAX_SAFE_INTEGER / 50)) invalid();
	const referenceId = params.get('reference_workspace_id') ?? '';
	// References are resolved from options locally; other IDs are validated by the API.
	if (params.has('reference_workspace_id') && (!/^[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}$/i.test(referenceId) || referenceId === '00000000-0000-0000-0000-000000000000')) invalid();
	return {
		mode, periodKind: periodKind as TimesheetPeriod['period'], period, groupPage, referenceId,
		owners: mode === 'personal' ? null : selection('owner_user_ids'),
		filters: { workspaces: selection('workspace_ids'), teams: selection('team_ids'), projects: selection('project_ids') }
	};
}

export function writeTimesheetLocation(url: URL, value: TimesheetLocation): URL {
	const result = new URL(url);
	for (const key of fields) result.searchParams.delete(key);
	const params = result.searchParams;
	params.set('mode', value.mode);
	params.set('period', value.period?.period ?? value.periodKind);
	if (value.period) {
		params.set('from', value.period.from);
		params.set('to', value.period.to);
	}
	if (value.referenceId) params.set('reference_workspace_id', value.referenceId);
	for (const [key, ids] of [
		['workspace_ids', value.filters.workspaces], ['team_ids', value.filters.teams], ['project_ids', value.filters.projects],
		...(value.mode === 'personal' ? [] : [['owner_user_ids', value.owners]])
	] as [string, TimesheetSelection][]) params.set(key, ids === null ? 'all' : [...new Set(ids)].join(','));
	params.set('page', String(value.groupPage));
	return result;
}
