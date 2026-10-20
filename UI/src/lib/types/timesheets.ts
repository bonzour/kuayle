export type TimesheetMode = 'personal' | 'person' | 'project' | 'workspace' | 'team';

// null means all; [] is an explicit empty selection. Projects also accept 'none'.
export type TimesheetSelection = string[] | null;
export interface TimesheetFilters {
	workspaces: TimesheetSelection;
	teams: TimesheetSelection;
	projects: TimesheetSelection;
}

export interface TimesheetWorkspaceOption {
	id: string;
	name: string;
	slug: string;
	efforts_enabled: boolean;
	timezone: string;
	today: string;
	can_administer: boolean;
	can_record_work: boolean;
}

export interface TimesheetTeamOption {
	id: string;
	workspace_id: string;
	name: string;
	key: string;
	icon: string | null;
	color: string | null;
}

export interface TimesheetProjectOption {
	id: string;
	workspace_id: string;
	name: string;
}

export interface TimesheetOptions {
	mode: TimesheetMode;
	available_modes: TimesheetMode[];
	workspaces: TimesheetWorkspaceOption[];
	teams: TimesheetTeamOption[];
	projects: TimesheetProjectOption[];
	scope: {
		workspace_ids: string[];
		team_ids: string[];
		project_ids: string[];
		include_no_project: boolean;
	};
}

export interface TimesheetPeriod {
	period: 'week' | 'month';
	from: string;
	to: string;
}

export interface TimesheetQuery extends TimesheetPeriod {
	mode: TimesheetMode;
	owners: TimesheetSelection;
	filters: TimesheetFilters;
	reference_workspace_id: string;
	page: number;
}

export interface TimesheetDayTotal { date: string; minutes: number }
export interface TimesheetGroup {
	key: string;
	type: 'person' | 'workspace' | 'team' | 'project' | 'no_project' | 'unavailable_project' | 'issue';
	id: string | null;
	name: string;
	workspace_id: string | null;
	workspace_slug: string | null;
	identifier?: string;
	team?: TimesheetTeamOption;
	person?: TimesheetWorklog['owner'];
	daily_totals: TimesheetDayTotal[];
	total_minutes: number;
	children: TimesheetGroup[];
}

export interface TimesheetReadResponse extends TimesheetPeriod {
	mode: TimesheetMode;
	days: string[];
	reference: TimesheetWorkspaceOption;
	workspaces: TimesheetWorkspaceOption[];
	scope: TimesheetOptions['scope'];
	daily_totals: TimesheetDayTotal[];
	total_minutes: number;
	total_count: number;
	page: number;
	per_page: number;
	has_more: boolean;
}

export interface TimesheetGrid extends TimesheetReadResponse {
	groups: TimesheetGroup[];
	people: TimesheetWorklog['owner'][];
	empty_reason: 'period' | 'filters' | null;
}

export interface TimesheetWorklog {
	id: string;
	issue_id: string;
	issue_identifier: string;
	issue_title: string;
	workspace: TimesheetWorkspaceOption;
	team: TimesheetTeamOption;
	owner: { id: string; available: boolean; name: string | null; display_name: string | null };
	operator: TimesheetWorklog['owner'];
	work_date: string;
	duration_minutes: number;
	description: string | null;
	created_at: string;
	updated_at: string;
}

export interface TimesheetWorklogs extends TimesheetReadResponse { data: TimesheetWorklog[] }

export interface TimesheetHistoryState {
	owners: TimesheetSelection;
	userId: string;
	slug: string;
	mode: TimesheetMode;
	filters: TimesheetFilters;
	period: TimesheetPeriod;
	referenceId: string;
	groupPage: number;
}
