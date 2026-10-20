<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { ChevronRight, Clock3, Pencil, Plus, Trash2 } from 'lucide-svelte';
	import { listEffortWorklogs } from '$lib/api/efforts';
	import { emitAppRefresh } from '$lib/api/refresh';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getCurrentWorkspace } from '$lib/features/workspaces/current-workspace.svelte';
	import type { Issue, IssueHistory } from '$lib/types/issue';
	import type { EffortUser, EffortWorklog, EffortWorklogPage } from '$lib/types/efforts';
	import { m } from '$lib/paraglide/messages.js';
	import { formatEffortMinutes } from './effort-duration';
	import EffortHistoryEntry from './EffortHistoryEntry.svelte';
	import { effortHistoryEntries } from './effort-history';
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
	let expanded = $state(true);
	const RECENT_COUNT = 3;
	let showAll = $state(false);
	let showAllHistory = $state(false);
	let recordsExpanded = $state(false);
	const events = $derived(effortHistoryEntries(history));
	const visibleEvents = $derived(showAllHistory ? events : events.slice(-RECENT_COUNT));
	const hiddenEvents = $derived(events.length - visibleEvents.length);
	const visibleWorklogs = $derived(showAll ? data?.data ?? [] : (data?.data ?? []).slice(0, RECENT_COUNT));
	const hiddenCount = $derived(Math.max((data?.data.length ?? 0) - RECENT_COUNT, 0));
	const canWrite = $derived(!!settings?.enabled && (admin || role === 'member') && !failed && !loading);
	let loadedIdentity = $state('');
	let selection = $state<{ mode: 'create' | 'update' | 'delete'; worklog: EffortWorklog | null; revision: number } | null>(null);
	let generation = 0;
	let activeIdentity = '';
	const headingId = $props.id();
	const userName = (user: EffortUser) => user.available ? user.display_name || user.name || m['efforts.worklogs.unavailable_user']() : m['efforts.worklogs.unavailable_user']();

	async function reload(page = 1) {
		const version = ++generation;
		const key = identity;
		const current = () => version === generation && key === identity;
		loading = true;
		try {
			const value = await listEffortWorklogs(slug, issue.identifier, page);
			if (!current()) return;
			if (data?.page !== value.page) showAll = false;
			data = value;
			loadedIdentity = key;
			failed = false;
			unavailable = false;
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

	function choose(mode: 'create' | 'update' | 'delete', worklog: EffortWorklog | null = null) {
		if (canWrite && data) selection = { mode, worklog, revision: data.revision };
	}

	$effect(() => {
		const key = identity;
		untrack(() => {
			if (activeIdentity !== key) {
				data = null;
				selection = null;
				failed = false;
				unavailable = false;
				expanded = true;
				showAll = false;
				showAllHistory = false;
				recordsExpanded = false;
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
				<div class="relative flex items-start gap-3 pb-2.5">
					<div class="relative z-10 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-bg)] text-[var(--color-text-tertiary)] ring-2 ring-[var(--color-bg)]" aria-hidden="true"><Clock3 size={12} /></div>
					<EffortHistoryEntry event={entry.effort!} />
				</div>
			{:else}<p class="text-xs text-[var(--color-text-tertiary)]">{m['issue.no_activity']()}</p>{/each}
			{#if showAllHistory && events.length > RECENT_COUNT}
				<button type="button" onclick={() => showAllHistory = false} class="relative z-10 mt-2 rounded-full border border-[var(--app-border)] bg-[var(--color-bg)] px-2.5 py-1 text-xs text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">{m['issue.show_less']()}</button>
			{/if}
		</div>
		<details bind:open={recordsExpanded} data-worklog-records class="border-t border-[var(--app-border)] pt-3">
			<summary class="mb-3 cursor-pointer text-xs font-medium text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">{m['efforts.worklogs.records']()} ({data.total_count})</summary>
		{#if !data.data.length}<p class="text-xs text-[var(--color-text-tertiary)]">{m['efforts.worklogs.empty']()}</p>{/if}
		<div class="relative">
			{#if visibleWorklogs.length > 0}<div class="absolute left-[9px] top-3 bottom-0 w-px bg-[var(--app-border)]" aria-hidden="true"></div>{/if}
			{#if hiddenCount > 0 && !showAll}
				<button type="button" onclick={() => showAll = true}
					class="relative z-10 mb-2 rounded-full border border-[var(--app-border)] bg-[var(--color-bg)] px-2.5 py-1 text-xs text-[var(--color-text-tertiary)] transition-colors hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-secondary)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">
					{m['efforts.worklogs.show_earlier']({ n: hiddenCount })}
				</button>
			{/if}
			<ul>
				{#each visibleWorklogs as worklog (worklog.id)}
					<li class="relative flex items-start gap-3 pb-2.5">
						<div class="relative z-10 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-bg)] text-[var(--color-text-tertiary)] ring-2 ring-[var(--color-bg)]" aria-hidden="true">
							<Clock3 size={12} />
						</div>
						<div class="min-w-0 flex-1 space-y-1">
							<div class="flex min-h-5 flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-[var(--color-text-tertiary)]">
								<strong class="text-[var(--color-text-secondary)]">{userName(worklog.owner)}</strong>
								<code class="shrink-0 rounded bg-[var(--color-bg-tertiary)] px-1 py-0.5 text-[11px] text-[var(--color-text-secondary)]">{formatEffortMinutes(worklog.duration_minutes)}</code>
								<span aria-hidden="true">&middot;</span>
								<time datetime={worklog.work_date}>{worklog.work_date}</time>
								{#if canWrite && (admin || worklog.owner.id === authState.user?.id)}
									<div class="ml-auto flex shrink-0 items-center gap-1 text-[var(--color-text-tertiary)]">
										<button type="button" aria-label={m['efforts.worklogs.edit']()} title={m['efforts.worklogs.edit']()} onclick={() => choose('update', worklog)}
											class="rounded p-1 transition-colors hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]"><Pencil size={12} /></button>
										<button type="button" aria-label={m['common.delete']()} title={m['common.delete']()} onclick={() => choose('delete', worklog)}
											class="rounded p-1 transition-colors hover:bg-[var(--color-bg-hover)] hover:text-destructive focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]"><Trash2 size={12} /></button>
									</div>
								{/if}
							</div>
							{#if worklog.owner.id !== worklog.operator.id}<p class="text-[11px] text-[var(--color-text-tertiary)]">{m['efforts.worklogs.inserted_by']({ name: userName(worklog.operator) })}</p>{/if}
							{#if worklog.description}<p class="whitespace-pre-wrap break-words text-xs text-[var(--color-text-secondary)]">{worklog.description}</p>{/if}
						</div>
					</li>
				{/each}
			</ul>
		</div>
		{#if hiddenCount > 0 && showAll}
			<button type="button" onclick={() => showAll = false}
				class="relative z-10 rounded-full border border-[var(--app-border)] bg-[var(--color-bg)] px-2.5 py-1 text-xs text-[var(--color-text-tertiary)] transition-colors hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-secondary)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--app-accent)]">
				{m['issue.show_less']()}
			</button>
		{/if}
		{#if data.page > 1 || data.has_more}
			<nav aria-label={m['efforts.worklogs.pages']()} class="flex items-center justify-between gap-3">
				<Button size="sm" variant="outline" disabled={loading || data.page === 1} onclick={() => reload(data!.page - 1)}>{m['efforts.worklogs.previous']()}</Button>
				<span class="text-xs">{m['efforts.worklogs.page']({ page: data.page })}</span>
				<Button size="sm" variant="outline" disabled={loading || !data.has_more} onclick={() => reload(data!.page + 1)}>{m['efforts.worklogs.next']()}</Button>
			</nav>
		{/if}
		</details>
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
