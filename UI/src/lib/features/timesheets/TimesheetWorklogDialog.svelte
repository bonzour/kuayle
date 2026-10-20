<script lang="ts">
	import { onMount, onDestroy, untrack } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import EffortWorklogDialog from '$lib/features/issues/EffortWorklogDialog.svelte';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getTimesheetOptions, getTimesheetIssues } from '$lib/api/timesheets';
	import { listEffortWorklogs } from '$lib/api/efforts';
	import type { EffortWorklog } from '$lib/types/efforts';
	import type { TimesheetIssues, TimesheetWorkspaceOption, TimesheetWorklogTarget } from '$lib/types/timesheets';
	import { allTimesheetFilters } from './selection';
	import { m } from '$lib/paraglide/messages.js';

	let { target, returnFocus, onclose, onsaved, onaccesschanged }: {
		target: TimesheetWorklogTarget; returnFocus: () => void;
		onclose: () => void; onsaved: (slug: string) => void; onaccesschanged: () => void;
	} = $props();
	const userId = untrack(() => authState.user!.id);
	const id = $props.id();
	let workspaces = $state<TimesheetWorkspaceOption[]>([]);
	let workspaceId = $state('');
	let search = $state('');
	let page = $state(1);
	let issues = $state<TimesheetIssues | null>(null);
	let loading = $state(true);
	let preparing = $state(false);
	let error = $state('');
	let editor = $state<{ workspace: TimesheetWorkspaceOption; identifier: string; revision: number; worklog: EffortWorklog | null } | null>(null);
	let generation = 0;
	let disposed = false;
	const current = (version: number) => !disposed && version === generation && userId === authState.user?.id;

	function fail(failure: unknown) {
		const status = (failure as { status?: number }).status;
		error = [401, 403, 404].includes(status ?? 0) ? m['efforts.planning.unavailable']() : m['timesheet.failed_load']();
		if ([401, 403, 404].includes(status ?? 0)) onaccesschanged();
	}

	async function initialize() {
		const version = ++generation;
		loading = true; error = '';
		try {
			const result = await getTimesheetOptions('personal', allTimesheetFilters());
			if (!current(version)) return;
			workspaces = result.workspaces.filter(w => w.can_record_work);
			if (target.workspaceId && target.identifier) {
				workspaceId = target.workspaceId;
				await prepare(target.identifier);
			}
		} catch (failure) { if (current(version)) fail(failure); }
		finally { if (current(version)) loading = false; }
	}

	async function loadIssues() {
		const version = ++generation;
		issues = null; error = ''; loading = true;
		try {
			const result = await getTimesheetIssues(workspaceId, search, page);
			if (!current(version)) return;
			issues = result;
			if (!result.workspace.can_record_work) error = m['timesheet.read_only']();
		} catch (failure) { if (current(version)) fail(failure); }
		finally { if (current(version)) loading = false; }
	}

	async function prepare(identifier: string) {
		const version = ++generation;
		preparing = true; error = '';
		try {
			const result = await getTimesheetOptions('personal', { ...allTimesheetFilters(), workspaces: [workspaceId] });
			if (!current(version)) return;
			const workspace = result.workspaces.find(w => w.id === workspaceId);
			if (!workspace?.can_record_work) { error = m['timesheet.read_only'](); onaccesschanged(); return; }
			let logs = await listEffortWorklogs(workspace.slug, identifier);
			if (!current(version)) return;
			let worklog = logs.data.find(w => w.id === target.worklogId);
			while (target.mode !== 'create' && !worklog && logs.has_more) {
				logs = await listEffortWorklogs(workspace.slug, identifier, logs.page + 1);
				if (!current(version)) return;
				worklog = logs.data.find(w => w.id === target.worklogId);
			}
			if (target.mode !== 'create' && (!worklog || (!workspace.can_administer && worklog.owner.id !== userId))) {
				error = m['efforts.worklogs.missing'](); return;
			}
			editor = { workspace, identifier, revision: logs.revision, worklog: worklog ?? null };
		} catch (failure) { if (current(version)) fail(failure); }
		finally { if (current(version)) { preparing = false; loading = false; } }
	}

	function changeWorkspace(value: string) {
		generation++;
		workspaceId = value; search = ''; page = 1; issues = null; error = '';
	}

	$effect(() => {
		void workspaceId; void search; void page;
		if (!workspaceId || target.identifier || editor) return;
		untrack(() => { generation++; issues = null; loading = true; });
		const timer = setTimeout(() => { void loadIssues(); }, 250);
		return () => { clearTimeout(timer); generation++; };
	});
	onMount(() => { void initialize(); });
	onDestroy(() => { disposed = true; generation++; });
</script>

{#if editor && authState.user?.id === userId}
	<EffortWorklogDialog slug={editor.workspace.slug} identifier={editor.identifier} mode={target.mode}
		worklog={editor.worklog} revision={editor.revision} {userId}
		userName={authState.user.display_name || authState.user.name}
		admin={editor.workspace.can_administer} canWrite={editor.workspace.can_record_work}
		initialDate={target.date} initialOwnerId={target.ownerId}
		workspaceLabel={`${editor.workspace.name} (${editor.workspace.slug})`}
		{returnFocus} {onclose} onsaved={() => onsaved(editor!.workspace.slug)} {onaccesschanged} />
{:else}
	<Dialog.Root open={true} onOpenChange={(open) => { if (!open && !preparing) onclose(); }}>
		<Dialog.Content showCloseButton={!preparing} class="max-h-[80vh] grid-rows-[auto_minmax(0,1fr)_auto]"
			onEscapeKeydown={(event) => { if (preparing) event.preventDefault(); }}
			onInteractOutside={(event) => { if (preparing) event.preventDefault(); }}
			onCloseAutoFocus={(event) => { event.preventDefault(); if (!editor) returnFocus(); }}>
			<Dialog.Header>
				<Dialog.Title>{m['efforts.worklogs.create']()}</Dialog.Title>
				<Dialog.Description>{m['timesheet.choose_destination']()}</Dialog.Description>
			</Dialog.Header>
			<div class="min-h-0 space-y-3 overflow-y-auto">
				{#if !target.identifier}
					<label for={`${id}-workspace`} class="block text-sm">{m['timesheet.workspaces']()}</label>
					<select id={`${id}-workspace`} value={workspaceId} disabled={preparing} onchange={(event) => changeWorkspace(event.currentTarget.value)} class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] p-2">
						<option value="">{m['timesheet.choose_workspace']()}</option>
						{#each workspaces as workspace}<option value={workspace.id}>{workspace.name} ({workspace.slug})</option>{/each}
					</select>
					{#if workspaceId}
						<label for={`${id}-search`} class="block text-sm">{m['timesheet.search_issues']()}</label>
						<input id={`${id}-search`} type="search" value={search} disabled={preparing} oninput={(event) => { search = event.currentTarget.value; page = 1; }} class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] p-2" />
					{/if}
				{/if}
				{#if error}<ErrorState message={error} retryLabel={m['timesheet.retry']()} onretry={() => target.identifier ? initialize() : workspaceId ? loadIssues() : initialize()} />
				{:else if loading || preparing}<LoadingState />
				{:else if issues}
					{#if issues.data.length === 0}<p class="text-sm">{m['timesheet.no_results']()}</p>{/if}
					<ul class="space-y-1" aria-label={m['timesheet.search_issues']()}>
						{#each issues.data as issue (issue.id)}
							<li><button type="button" class="w-full rounded-md border border-[var(--app-border)] p-2 text-left hover:bg-[var(--color-bg-secondary)] focus-visible:outline-2 focus-visible:outline-ring" onclick={() => prepare(issue.identifier)}><span class="block text-sm">{issue.identifier} · {issue.title}</span><span class="text-xs text-[var(--color-text-secondary)]">{issue.team.name}</span></button></li>
						{/each}
					</ul>
					{#if issues.total_count > issues.per_page}
						<div class="flex flex-wrap items-center justify-end gap-2">
							<span class="text-xs">{m['timesheet.page']({ page, pages: Math.ceil(issues.total_count / issues.per_page) })}</span>
							<Button variant="outline" size="sm" disabled={page <= 1} onclick={() => page--}>{m['timesheet.previous']()}</Button>
							<Button variant="outline" size="sm" disabled={!issues.has_more} onclick={() => page++}>{m['timesheet.next']()}</Button>
						</div>
					{/if}
				{:else if !workspaces.length}<p class="text-sm">{m['timesheet.read_only']()}</p>{/if}
			</div>
			<Dialog.Footer><Button variant="outline" disabled={preparing} onclick={onclose}>{m['common.cancel']()}</Button></Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>
{/if}
