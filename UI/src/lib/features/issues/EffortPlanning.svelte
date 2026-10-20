<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { ChevronRight } from 'lucide-svelte';
	import { getIssueEfforts } from '$lib/api/efforts';
	import { getIssue } from '$lib/api/issues';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getCurrentWorkspace } from '$lib/features/workspaces/current-workspace.svelte';
	import type { Issue } from '$lib/types/issue';
	import type { EffortSummary } from '$lib/types/efforts';
	import { m } from '$lib/paraglide/messages.js';
	import EffortPlanningField from './EffortPlanningField.svelte';
	import EffortBar from './EffortBar.svelte';
	import { formatEffortMinutes } from './effort-duration';

	let { issue, slug }: { issue: Issue; slug: string } = $props();
	const workspace = getCurrentWorkspace();
	const identity = $derived(`${slug}:${issue.id}:${authState.user?.id}`);
	const settings = $derived(workspace.efforts?.slug === slug && workspace.efforts.userId === authState.user?.id ? workspace.efforts.settings : null);
	const membership = $derived(workspace.membership?.workspace.slug === slug && workspace.membership.userId === authState.user?.id ? workspace.membership : null);
	let summary = $state<EffortSummary | null>(null);
	let currentIssue = $state<Issue | null>(null);
	let loading = $state(false);
	let failed = $state(false);
	let forbidden = $state(false);
	let generation = 0;
	let activeIdentity = '';
	let loadedIdentity = $state('');
	const headingId = $props.id();
	let expanded = $state(true);
	let totalsExpanded = $state(true);
	const ownScale = $derived(Math.max(summary?.own.estimate_minutes ?? 0, (summary?.own.spent_minutes ?? 0) + (summary?.own.remaining_minutes ?? 0), 1));
	const totalScale = $derived(Math.max(summary?.total.estimate_minutes ?? 0, (summary?.total.spent_minutes ?? 0) + (summary?.total.remaining_minutes ?? 0), 1));
	const showTotals = $derived(summary && loadedIdentity === identity && summary.has_visible_descendants && summary.total.spent_minutes > summary.own.spent_minutes);
	const terminal = $derived(currentIssue?.status_info
		? ['completed', 'cancelled'].includes(currentIssue.status_info.category)
		: ['done', 'completed', 'cancelled'].includes(currentIssue?.status ?? ''));
	const assigned = $derived(!!authState.user?.id && (currentIssue?.assignee_id === authState.user.id || !!currentIssue?.assignees?.some(user => user.id === authState.user?.id)));
	const hasPermission = $derived(!!membership && (['owner', 'admin'].includes(membership.workspace.current_user_role) || membership.workspace.current_user_role === 'member' && assigned));
	const hidden = $derived(settings?.enabled === false && summary?.has_data === false);
	const canEdit = $derived(!loading && !failed && !forbidden && !!settings?.enabled && hasPermission);

	async function reload(): Promise<EffortSummary | null> {
		const version = ++generation;
		const key = identity;
		const current = () => version === generation && key === identity;
		loading = true;
		try {
			const [value, fresh] = await Promise.all([getIssueEfforts(slug, issue.identifier), getIssue(slug, issue.identifier)]);
			if (!current()) return null;
			loadedIdentity = key;
			summary = value;
			currentIssue = fresh;
			failed = false;
			forbidden = false;
			return value;
		} catch (failure) {
			if (current()) {
				failed = true;
				const status = (failure as { status?: number })?.status;
				if (status === 401 || status === 403 || status === 404) {
					summary = null;
					currentIssue = null;
					forbidden = true;
				}
			}
			return null;
		} finally {
			if (current()) loading = false;
		}
	}

	$effect(() => {
		const key = identity;
		// Parent updates also cover local status/assignment changes before a socket refresh.
		void [issue.status, issue.status_id, issue.status_info?.category, issue.assignee_id, issue.assignees?.map(user => user.id).join(',')];
		untrack(() => {
			if (activeIdentity !== key) {
				summary = null;
				currentIssue = null;
				failed = false;
				forbidden = false;
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
		function issueUpdated(event: Event) {
			const detail = (event as CustomEvent).detail;
			if (detail?.id === issue.id || detail?.issue_id === issue.id || detail?.identifier === issue.identifier) void reload();
		}
		window.addEventListener('app:refresh', refresh);
		window.addEventListener('ws:issue-updated', issueUpdated);
		window.addEventListener('ws:reconnected', reload);
		return () => {
			generation++;
			window.removeEventListener('app:refresh', refresh);
			window.removeEventListener('ws:issue-updated', issueUpdated);
			window.removeEventListener('ws:reconnected', reload);
		};
	});
</script>

{#if !hidden}
<section class="rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)]" aria-labelledby={headingId}>
	<button id={headingId} type="button" onclick={() => expanded = !expanded}
		aria-expanded={expanded} aria-controls={`${headingId}-content`}
		class="flex w-full items-center gap-1.5 px-3 py-2.5 text-[11px] font-medium uppercase tracking-wider text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors">
		<ChevronRight size={12} class="transition-transform {expanded ? 'rotate-90' : ''}" />
		{m['efforts.planning.title']()}
	</button>
	<div id={`${headingId}-content`} hidden={!expanded} class="px-1.5 pb-2 space-y-0.5">
		{#if failed}
			<p role="alert" class="px-2 py-1.5 text-xs text-destructive">{m['efforts.planning.load_error']()}</p>
			<Button size="sm" variant="outline" onclick={() => reload()}>{m['settings.efforts.reload']()}</Button>
		{/if}
		{#if summary && loadedIdentity === identity}
			{#key identity}
				<EffortPlanningField kind="estimate" {slug} identifier={issue.identifier} {summary} {canEdit} {reload} scale={ownScale} />
				<div class="rounded-md px-2 py-2 hover:bg-[var(--color-bg-hover)] transition-colors">
					<div class="flex items-center justify-between gap-3">
						<span class="text-xs text-[var(--color-text-tertiary)]">{m['efforts.planning.spent']()}</span>
						<output aria-label={m['efforts.planning.spent']()} class="text-sm text-[var(--color-text-primary)]">{formatEffortMinutes(summary.own.spent_minutes)}</output>
					</div>
					<div class="mt-1.5"><EffortBar kind="spent" minutes={summary.own.spent_minutes} scale={ownScale} /></div>
				</div>
				<EffortPlanningField kind="remaining" {slug} identifier={issue.identifier} {summary} canEdit={canEdit && !terminal} {reload} scale={ownScale} />
			{/key}
			{#if !settings?.enabled}<p class="px-2 py-1.5 text-xs text-[var(--color-text-tertiary)]">{settings ? m['efforts.planning.disabled']() : m['settings.efforts.load_error']()}</p>
			{:else if membership && !hasPermission && !loading && !failed}<p class="px-2 py-1.5 text-xs text-[var(--color-text-tertiary)]">{m['efforts.planning.readonly']()}</p>{/if}
			{#if terminal}<p class="px-2 py-1.5 text-xs text-[var(--color-text-tertiary)]">{m['efforts.planning.closed']()}</p>{/if}
		{:else if !failed}
			<p role="status" class="px-2 py-1.5 text-xs">{m['settings.efforts.loading']()}</p>
		{/if}
	</div>
</section>
{#if showTotals && summary}
<section class="mt-3 rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)]" aria-labelledby={`${headingId}-totals`}>
	<button id={`${headingId}-totals`} type="button" onclick={() => totalsExpanded = !totalsExpanded}
		aria-expanded={totalsExpanded} aria-controls={`${headingId}-totals-content`}
		class="flex w-full items-center gap-1.5 px-3 py-2.5 text-[11px] font-medium uppercase tracking-wider text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)] transition-colors">
		<ChevronRight size={12} class="transition-transform {totalsExpanded ? 'rotate-90' : ''}" />
		{m['efforts.totals.title']()}
	</button>
	<div id={`${headingId}-totals-content`} hidden={!totalsExpanded} class="px-1.5 pb-2 space-y-0.5">
		{#each ([
			{ kind: 'estimate', label: m['efforts.planning.estimate'](), value: summary.total.estimate_minutes, partial: summary.total.estimate_partial },
			{ kind: 'spent', label: m['efforts.planning.spent'](), value: summary.total.spent_minutes, partial: false },
			{ kind: 'remaining', label: m['efforts.planning.remaining'](), value: summary.total.remaining_minutes, partial: summary.total.remaining_partial }
		] as const) as total}
			<div class="rounded-md px-2 py-2">
				<div class="flex items-center justify-between gap-3">
					<span class="text-xs text-[var(--color-text-tertiary)]">{total.label}</span>
					<output aria-label={total.label} class="text-right text-sm text-[var(--color-text-primary)]">{total.value === null ? m['efforts.planning.unestimated']() : formatEffortMinutes(total.value)}{#if total.partial && total.value !== null}<span class="ml-1 text-[11px] text-[var(--color-text-tertiary)]">({m['efforts.totals.partial']()})</span>{/if}</output>
				</div>
				<div class="mt-1.5"><EffortBar kind={total.kind} minutes={total.value} scale={totalScale} /></div>
			</div>
		{/each}
	</div>
</section>
{/if}
{/if}
