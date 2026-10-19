import { api } from './client';
import type { CreateEffortWorklogRequest, UpdateEffortWorklogRequest, DeleteEffortWorklogRequest, EffortUser, EffortWorklogPage, EffortMutationReceipt, EffortSettings, EffortSummary, SetEffortEstimateRequest, SetEffortRemainingRequest, SetEffortSettingsRequest } from '$lib/types/efforts';

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

export function listEffortWorklogs(slug: string, identifier: string, page = 1): Promise<EffortWorklogPage> {
	return api.get(`${issue(slug, identifier)}/worklogs?page=${page}&per_page=50`);
}

export function listEffortWorklogOwners(slug: string, identifier: string): Promise<EffortUser[]> {
	return api.get(`${issue(slug, identifier)}/worklog-owners`);
}

export function createEffortWorklog(slug: string, identifier: string, request: CreateEffortWorklogRequest): Promise<EffortMutationReceipt> {
	return api.post(`${issue(slug, identifier)}/worklogs`, request);
}

export function updateEffortWorklog(slug: string, identifier: string, id: string, request: UpdateEffortWorklogRequest): Promise<EffortMutationReceipt> {
	return api.put(`${issue(slug, identifier)}/worklogs/${encodeURIComponent(id)}`, request);
}

export function deleteEffortWorklog(slug: string, identifier: string, id: string, request: DeleteEffortWorklogRequest): Promise<EffortMutationReceipt> {
	return api.deleteWithBody(`${issue(slug, identifier)}/worklogs/${encodeURIComponent(id)}`, request);
}

export function setEffortEstimate(slug: string, identifier: string, request: SetEffortEstimateRequest): Promise<EffortMutationReceipt> {
	return api.put(`${issue(slug, identifier)}/estimate`, request);
}

export function setEffortRemaining(slug: string, identifier: string, request: SetEffortRemainingRequest): Promise<EffortMutationReceipt> {
	return api.put(`${issue(slug, identifier)}/remaining`, request);
}
