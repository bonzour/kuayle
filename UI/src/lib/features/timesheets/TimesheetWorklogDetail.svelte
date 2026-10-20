<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getTimesheetWorklogs } from '$lib/api/timesheets';
	import type { TimesheetGroup, TimesheetQuery, TimesheetWorklog, TimesheetWorklogs } from '$lib/types/timesheets';
	import { formatTimesheetDate, formatTimesheetMinutes } from './period';
	import { personLabel } from './people';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let { query, group, date, returnFocus, onclose, onaccesschanged }: {
		query: TimesheetQuery; group: TimesheetGroup; date: string; returnFocus: HTMLElement | null;
		onclose: () => void; onaccesschanged: () => void;
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
			if (current()) result = response;
		} catch (error) {
			if (!current()) return;
			if ([401, 403, 404].includes((error as { status?: number }).status ?? 0)) onaccesschanged();
			else failed = true;
		} finally { if (current()) loading = false; }
	}
	$effect(() => { void query; void group; void date; void page; untrack(() => { void load(); }); });
	onDestroy(() => { generation++; });
</script>

<Dialog.Root open={true} onOpenChange={(open) => { if (!open) onclose(); }}>
	<Dialog.Content class="max-h-[80vh] sm:max-w-2xl" onCloseAutoFocus={(event) => { event.preventDefault(); if (returnFocus?.isConnected) returnFocus.focus(); }}>
		<Dialog.Header>
			<Dialog.Title>{m['timesheet.worklog_details']()}</Dialog.Title>
			<Dialog.Description>{group.identifier} · {formatTimesheetDate(date, getLocale(), { year: 'numeric' })}</Dialog.Description>
		</Dialog.Header>
		<div class="min-h-0 space-y-4 overflow-y-auto">
			{#if loading}<LoadingState />
			{:else if failed}<ErrorState message={m['timesheet.failed_load']()} onretry={load} />
			{:else if result}
				<p class="text-sm font-medium" role="status">{m['timesheet.total']()}: {formatTimesheetMinutes(result.total_minutes)} · {m['timesheet.registration_count']({ count: result.total_count })}</p>
				{#if result.data.length === 0}<p class="text-sm text-[var(--color-text-secondary)]">{m['timesheet.no_results']()}</p>{/if}
				<ul class="space-y-3">
					{#each result.data as worklog (worklog.id)}
						<li class="space-y-2 rounded-lg border border-[var(--app-border)] p-3">
							<div class="flex flex-wrap justify-between gap-2"><a class="rounded font-medium hover:underline focus-visible:outline-2 focus-visible:outline-ring" href={`/${worklog.workspace.slug}/issue/${worklog.issue_identifier}`}>{worklog.issue_identifier} · {worklog.issue_title}</a><strong>{formatTimesheetMinutes(worklog.duration_minutes)}</strong></div>
							<p class="text-xs text-[var(--color-text-secondary)]">{worklog.workspace.name} ({worklog.workspace.slug}) · {worklog.team.name} · {worklog.work_date}</p>
							<dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm"><dt>{m['timesheet.owner']()}</dt><dd>{ownerName(worklog.owner)}</dd><dt>{m['timesheet.operator']()}</dt><dd>{ownerName(worklog.operator)}</dd></dl>
							{#if worklog.description}<p class="whitespace-pre-wrap break-words text-sm">{worklog.description}</p>{/if}
							<p class="text-xs text-[var(--color-text-secondary)]">{worklog.workspace.timezone} · {m['timesheet.today']()}: {worklog.workspace.today} · {m['timesheet.reference']()}: {result.reference.timezone}</p>
						</li>
					{/each}
				</ul>
				{#if result.total_count > result.per_page}
					<div class="flex flex-wrap items-center justify-end gap-2" aria-label={m['timesheet.detail_pages']()}>
						<span class="text-xs">{m['timesheet.page']({ page: result.page, pages: Math.ceil(result.total_count / result.per_page) })}</span>
						<Button variant="outline" size="sm" disabled={page <= 1} onclick={() => page--}>{m['timesheet.previous']()}</Button>
						<Button variant="outline" size="sm" disabled={!result.has_more} onclick={() => page++}>{m['timesheet.next']()}</Button>
					</div>
				{/if}
			{/if}
		</div>
		<Dialog.Footer><Button variant="outline" size="sm" onclick={onclose}>{m['timesheet.close']()}</Button></Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
