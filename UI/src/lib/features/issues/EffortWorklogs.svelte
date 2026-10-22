<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { ChevronRight, Clock3, Plus } from 'lucide-svelte';
	import { listEffortWorklogs } from '$lib/api/efforts';
	import { emitAppRefresh } from '$lib/api/refresh';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getCurrentWorkspace } from '$lib/features/workspaces/current-workspace.svelte';
	import type { Issue, IssueHistory } from '$lib/types/issue';
	import type { EffortWorklog, EffortWorklogPage } from '$lib/types/efforts';
	import { m } from '$lib/paraglide/messages.js';
	import EffortHistoryEntry from './EffortHistoryEntry.svelte';
	import { effortHistoryEntries } from './effort-history';
	import WorklogActions from './WorklogActions.svelte';
	import EffortWorklogDialog from './EffortWorklogDialog.svelte';

	let { issue, slug, history = [] }: { issue: Issue; slug: string; history?: IssueHistory[] } = $props();
	const workspace = getCurrentWorkspace();
	const identity = $derived(`${slug}:${issue.id}:${authState.user?.id}`);
	const settings = $derived(workspace.efforts?.slug === slug && workspace.efforts.userId === authState.user?.id ? workspace.efforts.settings : null);
	const membership = $derived(workspace.membership?.workspace.slug === slug && workspace.membership.userId === authState.user?.id ? workspace.membership : null);
	const role = $derived(membership?.workspace.current_user_role);
	const admin = $derived(role === 'owner' || role === 'admin');
	let data = $state<EffortWorklogPage | null>(null);
	let loading = $state(false);
	let failed = $state(false);
	let unavailable = $state(false);
	let expanded = $state(false);
	const RECENT_COUNT = 1;
	let showAllHistory = $state(false);
	const events = $derived(effortHistoryEntries(history));
	const visibleEvents = $derived(showAllHistory ? events : events.slice(-RECENT_COUNT));
	const hiddenEvents = $derived(events.length - visibleEvents.length);
	const latestWorklogEvents = $derived.by(() => {
		const latest = new Map<string, { id: string; revision: number }>();
		for (const entry of events) {
			const snapshot = entry.effort!.after ?? entry.effort!.before;
			if (snapshot && 'worklog' in snapshot && (latest.get(snapshot.worklog.id)?.revision ?? -1) < entry.effort!.revision) {
				latest.set(snapshot.worklog.id, { id: entry.id, revision: entry.effort!.revision });
			}
		}
		return latest;
	});
	const canWrite = $derived(!!settings?.enabled && (admin || role === 'member') && !failed && !loading);
	let loadedIdentity = $state('');
	let selection = $state<{ mode: 'create' | 'update' | 'delete'; worklog: EffortWorklog | null; revision: number } | null>(null);
	let generation = 0;
	let activeIdentity = '';
	const headingId = $props.id();

	async function reload(worklogId?: string) {
		const version = ++generation;
		const key = identity;
		const current = () => version === generation && key === identity;
		loading = true;
		try {
			let value = await listEffortWorklogs(slug, issue.identifier);
			if (!current()) return;
			while (worklogId && !value.data.some(row => row.id === worklogId) && value.has_more) {
				value = await listEffortWorklogs(slug, issue.identifier, value.page + 1);
				if (!current()) return;
			}
			const worklog = value.data.find(row => row.id === worklogId);
			if (worklogId && !worklog) throw new Error('Work log no longer available');
			data = value;
			loadedIdentity = key;
			failed = false;
			unavailable = false;
			return worklog;
		} catch (failure) {
			if (!current()) return;
			failed = true;
			if ([401, 403, 404].includes((failure as { status?: number }).status ?? 0)) {
				data = null;
				selection = null;
				unavailable = true;
			}
		} finally {
			if (current()) loading = false;
		}
	}

	async function choose(mode: 'create' | 'update' | 'delete', worklogId?: string) {
		if (!canWrite) return;
		const worklog = worklogId ? await reload(worklogId) : null;
		if (worklogId && !worklog) return;
		if (canWrite && data) selection = { mode, worklog: worklog ?? null, revision: data.revision };
	}

	$effect(() => {
		const key = identity;
		untrack(() => {
			if (activeIdentity !== key) {
				data = null;
				selection = null;
				failed = false;
				unavailable = false;
				expanded = false;
				showAllHistory = false;
			}
			activeIdentity = key;
			void reload();
		});
		return () => { generation++; };
	});

	onMount(() => {
		function refresh(event: Event) {
			const detail = (event as CustomEvent<{ slug?: string; resources?: string[] }>).detail;
			if (detail?.slug && detail.slug !== slug) return;
			if (!detail?.resources?.length || detail.resources.some(resource => ['issues', 'workspace', 'members', 'teams'].includes(resource))) void reload();
		}
		const reconnect = () => { void reload(); };
		window.addEventListener('app:refresh', refresh);
		window.addEventListener('ws:reconnected', reconnect);
		return () => {
			generation++;
			window.removeEventListener('app:refresh', refresh);
			window.removeEventListener('ws:reconnected', reconnect);
		};
	});
</script>

<section class="mt-6 border-t border-[var(--app-border)] pt-4" aria-labelledby={headingId} aria-busy={loading}>
	<div class="flex items-center justify-between gap-3 {expanded ? 'mb-3' : ''}">
		<h3 id={headingId} class="text-xs font-medium uppercase tracking-wide text-[var(--color-text-tertiary)]">
			<button type="button" onclick={() => expanded = !expanded} aria-expanded={expanded} aria-controls={`${headingId}-content`}
				class="flex items-center gap-1.5 rounded-sm text-inherit uppercase tracking-wide outline-none transition-colors hover:text-[var(--color-text-secondary)] focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">
				<ChevronRight size={12} class="transition-transform {expanded ? 'rotate-90' : ''}" aria-hidden="true" />
				{m['efforts.worklogs.title']()}
			</button>
		</h3>
		{#if canWrite && data}
			<button type="button" onclick={() => { expanded = true; choose('create'); }}
				class="inline-flex items-center gap-1 rounded-full border border-[var(--app-border)] bg-[var(--color-bg)] px-2.5 py-1 text-xs text-[var(--color-text-tertiary)] transition-colors hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-secondary)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">
				<Plus size={12} aria-hidden="true" />{m['efforts.worklogs.create']()}
			</button>
		{/if}
	</div>
	<div id={`${headingId}-content`} hidden={!expanded} class="space-y-3">
	{#if failed}
		<p role="alert" class="text-xs text-destructive">{unavailable ? m['efforts.planning.unavailable']() : m['efforts.worklogs.load_error']()}</p>
		<Button size="sm" variant="outline" onclick={() => reload()}>{m['settings.efforts.reload']()}</Button>
	{/if}
	{#if data && loadedIdentity === identity}
		{#if !settings?.enabled}<p class="text-xs text-[var(--color-text-tertiary)]">{settings ? m['efforts.planning.disabled']() : m['settings.efforts.load_error']()}</p>
		{:else if role === 'guest'}<p class="text-xs text-[var(--color-text-tertiary)]">{m['efforts.worklogs.readonly']()}</p>{/if}
		<div class="relative">
			{#if visibleEvents.length}<div class="absolute left-[9px] top-3 bottom-0 w-px bg-[var(--app-border)]" aria-hidden="true"></div>{/if}
			{#if hiddenEvents > 0}
				<button type="button" onclick={() => showAllHistory = true} class="relative z-10 mb-2 rounded-full border border-[var(--app-border)] bg-[var(--color-bg)] px-2.5 py-1 text-xs text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-secondary)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">{m['issue.show_earlier']({ n: hiddenEvents })}</button>
			{/if}
			{#each visibleEvents as entry (entry.id)}
				{@const snapshot = entry.effort!.after}
				{@const worklog = snapshot && 'worklog' in snapshot && latestWorklogEvents.get(snapshot.worklog.id)?.id === entry.id ? snapshot.worklog : null}
				<div data-worklog-id={worklog?.id} class="relative flex items-start gap-3 pb-2.5">
					<div class="relative z-10 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-bg)] text-[var(--color-text-tertiary)] ring-2 ring-[var(--color-bg)]" aria-hidden="true"><Clock3 size={12} /></div>
					<EffortHistoryEntry event={entry.effort!}>
						{#snippet actions()}
							{#if worklog && canWrite && (admin || worklog.owner_user_id === authState.user?.id)}
								<WorklogActions onedit={() => choose('update', worklog.id)} ondelete={() => choose('delete', worklog.id)} />
							{/if}
						{/snippet}
					</EffortHistoryEntry>
				</div>
			{:else}<p class="text-xs text-[var(--color-text-tertiary)]">{m['issue.no_activity']()}</p>{/each}
			{#if showAllHistory && events.length > RECENT_COUNT}
				<button type="button" onclick={() => showAllHistory = false} class="relative z-10 mt-2 rounded-full border border-[var(--app-border)] bg-[var(--color-bg)] px-2.5 py-1 text-xs text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">{m['issue.show_less']()}</button>
			{/if}
		</div>
	{:else if loading}<p role="status" class="text-xs">{m['settings.efforts.loading']()}</p>{/if}
	</div>
</section>

{#if selection && authState.user}
	{#key identity}
		<EffortWorklogDialog {slug} identifier={issue.identifier} {admin} canWrite={!!settings?.enabled && (admin || role === 'member')}
			userId={authState.user.id} userName={authState.user.display_name || authState.user.name}
			mode={selection.mode} worklog={selection.worklog} revision={selection.revision}
			onclose={() => selection = null} onsaved={() => { selection = null; emitAppRefresh(['issues'], slug); }} />
	{/key}
{/if}
