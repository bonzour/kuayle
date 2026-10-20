import type { PaginatedResponse } from './common';

/** Calendar date in YYYY-MM-DD form; never pass through Date/toISOString. */
export type EffortDate = string;

export interface EffortSettings {
	enabled: boolean;
	timezone: string;
	revision: number;
	today: EffortDate;
}

export interface EffortValues {
	estimate_minutes: number | null;
	spent_minutes: number;
	remaining_minutes: number | null;
}

export interface EffortTotals {
	estimate_minutes: number | null;
	spent_minutes: number;
	remaining_minutes: number;
	estimate_partial: boolean;
	remaining_partial: boolean;
}

export interface EffortSummary {
 has_data: boolean;
 has_visible_descendants: boolean;
	revision: number;
	own: EffortValues;
	total: EffortTotals;
}

export interface EffortUser {
	id: string;
	available: boolean;
	name: string | null;
	display_name: string | null;
}

export interface EffortWorklog {
	id: string;
	issue_id: string;
	owner: EffortUser;
	operator: EffortUser;
	work_date: EffortDate;
	duration_minutes: number;
	description: string | null;
	created_at: string;
	updated_at: string;
}

export interface EffortWorklogPage extends PaginatedResponse<EffortWorklog> {
	revision: number;
}

// Direct issue writes require a session or explicit read/write Efforts scopes.
export interface EffortMutationRequest {
	request_id: string;
	expected_revision: number;
}

export interface SetEffortEstimateRequest extends EffortMutationRequest {
	estimate_minutes: number | null;
}

export interface SetEffortRemainingRequest extends EffortMutationRequest {
	remaining_minutes: number;
}

export interface UpdateEffortWorklogRequest extends EffortMutationRequest {
	work_date: EffortDate;
	duration_minutes: number;
	description: string | null;
}

export interface CreateEffortWorklogRequest extends UpdateEffortWorklogRequest {
	owner_user_id: string;
	end_date?: EffortDate;
	skip_weekends?: boolean;
}

export type DeleteEffortWorklogRequest = EffortMutationRequest;

export interface SetEffortSettingsRequest extends EffortMutationRequest {
	enabled: boolean;
	timezone: string;
}

export interface EffortMutationReceipt {
	request_id: string;
	resource_id: string;
	revision: number;
}

export type EffortActor =
	| { kind: 'user'; user_id: string }
	| { kind: 'system'; system_source: string };

export type EffortSnapshot =
	| { planning: Pick<EffortValues, 'estimate_minutes' | 'remaining_minutes'> }
	| { worklog: Omit<EffortWorklog, 'issue_id' | 'owner' | 'operator'> & { owner_user_id: string; created_by_user_id: string } };

export interface EffortEvent {
 users: EffortUser[];
	id: string;
	issue_id: string;
	event_type: 'estimate_set' | 'remaining_set' | 'worklog_created' | 'worklog_updated' | 'worklog_deleted';
	actor: EffortActor;
	cause: 'manual' | 'worklog' | 'closure' | 'reactivation';
	snapshot_version: 1;
	before: EffortSnapshot | null;
	after: EffortSnapshot | null;
	revision: number;
	created_at: string;
}

export type EffortErrorCode =
	| 'EFFORT_REVISION_CONFLICT'
	| 'EFFORT_REQUEST_CONFLICT'
	| 'EFFORT_DAILY_LIMIT'
	| 'EFFORT_DISABLED';
