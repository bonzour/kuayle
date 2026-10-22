<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import WorklogActions from '$lib/features/issues/WorklogActions.svelte';
	import ResultPagination from '$lib/components/shared/ResultPagination.svelte';
	import { X } from 'lucide-svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getTimesheetWorklogs } from '$lib/api/timesheets';
	import type { TimesheetGroup, TimesheetQuery, TimesheetWorklog, TimesheetWorklogs, TimesheetWorklogTarget } from '$lib/types/timesheets';
	import { formatTimesheetDate, formatTimesheetHours, formatTimesheetDays } from './period';
	import { personLabel } from './people';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let { query, group, date, returnFocus, fallbackFocus, refresh = 0, suspended = false, onaction, onclose, onaccesschanged }: {
		query: TimesheetQuery; group: TimesheetGroup; date: string; returnFocus: HTMLElement | null;
		fallbackFocus: () => void; onclose: () => void; onaccesschanged: () => void;
		refresh?: number; suspended?: boolean; onaction: (target: TimesheetWorklogTarget) => void;
	} = $props();
	let result = $state<TimesheetWorklogs | null>(null);
	let loading = $state(true);
	let failed = $state(false);
	let page = $state(1);
	let generation = 0;
	const ownerName = (user: TimesheetWorklog['owner']) => personLabel(user, m['timesheet.unavailable_user']());

	async function load() {
		const id = ++generation;
		const userId = authState.user?.id;
		const current = () => id === generation && userId === authState.user?.id;
		loading = true; failed = false; result = null;
		try {
			const response = await getTimesheetWorklogs({ ...query, page }, group.id!, date);
			if (!current()) return;
			const lastPage = Math.max(1, Math.ceil(response.total_count / response.per_page));
			if (page > lastPage) { page = lastPage; return; }
			result = response;
		} catch (error) {
			if (!current()) return;
			if ([401, 403, 404].includes((error as { status?: number }).status ?? 0)) onaccesschanged();
			else failed = true;
		} finally { if (current()) loading = false; }
	}
	$effect.pre(() => {
		void query; void group; void date; void page; void refresh;
		const paused = suspended;
		untrack(() => { generation++; result = null; loading = true; if (!paused) void load(); });
		return () => { generation++; };
	});
	onDestroy(() => { generation++; });
</script>

<Dialog.Root open={true} onOpenChange={(open) => { if (!open) onclose(); }}>
	<Dialog.Content data-timesheet-detail showCloseButton={false} class="max-h-[80vh] grid-rows-[auto_minmax(0,1fr)_auto] break-words sm:max-w-2xl" onCloseAutoFocus={(event) => { event.preventDefault(); if (returnFocus?.isConnected) returnFocus.focus(); else fallbackFocus(); }}>
		<Dialog.Close>
			{#snippet child({ props })}
				<Button {...props} variant="ghost" size="icon-sm" class="absolute right-2 top-2" aria-label={m['timesheet.close']()}><X size={16} /></Button>
			{/snippet}
		</Dialog.Close>
		<Dialog.Header class="pr-6">
			<Dialog.Title>{m['timesheet.worklog_details']()}</Dialog.Title>
			<Dialog.Description>{group.identifier} · {formatTimesheetDate(date, getLocale(), { year: 'numeric' })}</Dialog.Description>
		</Dialog.Header>
		<div aria-busy={loading} class="min-h-0 space-y-4 overflow-y-auto">
			{#if loading}<LoadingState variant="compact" />
			{:else if failed}<ErrorState variant="compact" message={m['timesheet.failed_load']()} retryLabel={m['timesheet.retry']()} onretry={load} />
			{:else if result}
				<p class="text-sm font-medium" role="status">{m['timesheet.total']()}: {formatTimesheetDays(result.total_minutes, getLocale())} · {m['timesheet.registration_count']({ count: result.total_count })}</p>
				{#if result.workspaces.some(w => w.id === group.workspace_id && w.can_record_work && date <= w.today)}
					<Button size="sm" onclick={() => onaction({ mode: 'create', workspaceId: group.workspace_id!, identifier: group.identifier!, date, ownerId: query.mode === 'personal' ? authState.user?.id : query.owners?.[0] })}>{m['efforts.worklogs.create']()}</Button>
				{/if}
				{#if result.data.length === 0}<p class="text-sm text-[var(--color-text-secondary)]">{m['timesheet.no_results']()}</p>{/if}
				<details class="text-xs text-muted-foreground"><summary class="min-h-8 cursor-pointer">{m['controls.scope_details']()}</summary>{#each result.workspaces as workspace}<p>{workspace.name} ({workspace.slug}) · {workspace.timezone}</p>{/each}<p>{m['timesheet.reference']()}: {result.reference.timezone}</p></details>
				<ul class="space-y-3">
					{#each result.data as worklog (worklog.id)}
						<li class="space-y-2 rounded-lg border border-[var(--app-border)] p-3">
							<div class="flex flex-wrap justify-between gap-2"><a class="min-w-0 max-w-full rounded font-medium hover:underline focus-visible:outline-2 focus-visible:outline-ring" href={`/${worklog.workspace.slug}/issue/${worklog.issue_identifier}`}>{worklog.issue_identifier} · {worklog.issue_title}</a><strong>{formatTimesheetHours(worklog.duration_minutes, getLocale())}</strong></div>
							<p class="text-xs text-[var(--color-text-secondary)]">{worklog.workspace.name} ({worklog.workspace.slug}) · {worklog.team.name} · {formatTimesheetDate(worklog.work_date, getLocale(), { year: 'numeric' })}</p>
							<dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-sm"><dt>{m['timesheet.owner']()}</dt><dd>{ownerName(worklog.owner)}</dd><dt>{m['timesheet.operator']()}</dt><dd>{ownerName(worklog.operator)}</dd></dl>
							{#if worklog.description}<p class="whitespace-pre-wrap break-words text-sm">{worklog.description}</p>{/if}

							{#if worklog.workspace.can_record_work && (worklog.workspace.can_administer || worklog.owner.id === authState.user?.id)}
								<WorklogActions onedit={() => onaction({ mode: 'update', workspaceId: worklog.workspace.id, identifier: worklog.issue_identifier, worklogId: worklog.id })} ondelete={() => onaction({ mode: 'delete', workspaceId: worklog.workspace.id, identifier: worklog.issue_identifier, worklogId: worklog.id })} />
							{/if}
						</li>
					{/each}
				</ul>
				<ResultPagination {page} perPage={result.per_page} total={result.total_count} label={m['timesheet.detail_pages']()} onchange={(value) => page = value} />
			{/if}
		</div>
		<Dialog.Footer><Button variant="outline" size="sm" onclick={onclose}>{m['timesheet.close']()}</Button></Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
