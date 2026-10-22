<script lang="ts">
	import { onMount, onDestroy, untrack } from 'svelte';
	import IssueSearchDialog from '$lib/features/issues/IssueSearchDialog.svelte';
	import FormSelect from '$lib/components/shared/FormSelect.svelte';
	import ResultPagination from '$lib/components/shared/ResultPagination.svelte';
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
	<IssueSearchDialog title={m['efforts.worklogs.create']()} description={m['timesheet.choose_destination']()} query={search} onquery={(value) => { search = value; page = 1; }}
		items={issues?.data.map(issue => ({ ...issue, subtitle: issue.team.name })) ?? []} onselect={(id) => { const issue = issues?.data.find(issue => issue.id === id); if (issue) void prepare(issue.identifier); }}
		loading={loading || preparing} error={error || (!loading && !workspaces.length ? m['timesheet.read_only']() : '')} disabled={preparing} searchable={!!workspaceId && !target.identifier}
		onretry={() => target.identifier ? initialize() : workspaceId ? loadIssues() : initialize()} {onclose} returnFocus={() => { if (!editor) returnFocus(); }}>
		{#snippet context()}
			{#if !target.identifier}<FormSelect id={`${id}-workspace`} label={m['timesheet.workspaces']()} value={workspaceId} options={workspaces.map(workspace => ({ id: workspace.id, name: `${workspace.name} (${workspace.slug})` }))} placeholder={m['timesheet.choose_workspace']()} disabled={preparing} onchange={changeWorkspace} />{/if}
		{/snippet}
		{#snippet footer()}
			{#if issues}<ResultPagination {page} perPage={issues.per_page} total={issues.total_count} label={m['controls.issue_search']()} onchange={(value) => page = value} />{/if}
		{/snippet}
	</IssueSearchDialog>
{/if}
