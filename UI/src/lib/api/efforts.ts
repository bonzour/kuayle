import { api } from './client';
import type { PaginatedResponse } from '$lib/types/common';
import type { EffortMutationReceipt, EffortSettings, EffortSummary, EffortWorklog, SetEffortEstimateRequest, SetEffortRemainingRequest, SetEffortSettingsRequest } from '$lib/types/efforts';

const base = (slug: string) => `/api/workspaces/${encodeURIComponent(slug)}`;
const issue = (slug: string, identifier: string) => `${base(slug)}/issues/${encodeURIComponent(identifier)}/efforts`;

export function getEffortSettings(slug: string): Promise<EffortSettings> {
	return api.get(`${base(slug)}/efforts/settings`);
}

export function setEffortSettings(slug: string, request: SetEffortSettingsRequest): Promise<EffortMutationReceipt> {
	return api.put(`${base(slug)}/efforts/settings`, request);
}

export function getIssueEfforts(slug: string, identifier: string): Promise<EffortSummary> {
	return api.get(issue(slug, identifier));
}

export function listEffortWorklogs(slug: string, identifier: string, page = 1): Promise<PaginatedResponse<EffortWorklog>> {
	return api.get(`${issue(slug, identifier)}/worklogs?page=${page}&per_page=50`);
}

export function setEffortEstimate(slug: string, identifier: string, request: SetEffortEstimateRequest): Promise<EffortMutationReceipt> {
	return api.put(`${issue(slug, identifier)}/estimate`, request);
}

export function setEffortRemaining(slug: string, identifier: string, request: SetEffortRemainingRequest): Promise<EffortMutationReceipt> {
	return api.put(`${issue(slug, identifier)}/remaining`, request);
}
