import { getContext, setContext } from 'svelte';
import type { Workspace } from '$lib/types/workspace';
import type { EffortSettings } from '$lib/types/efforts';
import { getEffortSettings } from '$lib/api/efforts';

const KEY = Symbol('current-workspace');

class CurrentWorkspace {
	// A role is valid only for the user and workspace that were loaded together.
	membership = $state<{ workspace: Workspace; userId: string } | null>(null);
	efforts = $state<{ slug: string; userId: string; settings: EffortSettings } | null>(null);
	effortsFailed = $state(false);
	private effortGeneration = 0;

	async loadEfforts(slug: string) {
		if (this.membership?.workspace.slug !== slug) return null;
		const generation = ++this.effortGeneration;
		const userId = this.membership?.userId;
		const current = () => generation === this.effortGeneration && !!userId &&
			this.membership?.userId === userId && this.membership.workspace.slug === slug;
		try {
			const settings = await getEffortSettings(slug);
			if (!current()) return null;
			this.efforts = { slug, userId: userId!, settings };
			this.effortsFailed = false;
			return settings;
		} catch {
			if (current()) {
				this.efforts = null;
				this.effortsFailed = true;
			}
			return null;
		}
	}

	clearEfforts() {
		this.effortGeneration++;
		this.efforts = null;
		this.effortsFailed = false;
	}
}

export function setCurrentWorkspace() {
	return setContext(KEY, new CurrentWorkspace());
}

export function getCurrentWorkspace() {
	return getContext<CurrentWorkspace>(KEY);
}
