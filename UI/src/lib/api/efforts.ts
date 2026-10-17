import { api } from './client';
import type { PaginatedResponse } from '$lib/types/common';
import type { EffortSettings, EffortSummary, EffortWorklog } from '$lib/types/efforts';

const base = (slug: string) => `/api/workspaces/${encodeURIComponent(slug)}`;
const issue = (slug: string, identifier: string) => `${base(slug)}/issues/${encodeURIComponent(identifier)}/efforts`;

export function getEffortSettings(slug: string): Promise<EffortSettings> {
	return api.get(`${base(slug)}/efforts/settings`);
}

export function getIssueEfforts(slug: string, identifier: string): Promise<EffortSummary> {
	return api.get(issue(slug, identifier));
}

export function listEffortWorklogs(slug: string, identifier: string, page = 1): Promise<PaginatedResponse<EffortWorklog>> {
	return api.get(`${issue(slug, identifier)}/worklogs?page=${page}&per_page=50`);
}
