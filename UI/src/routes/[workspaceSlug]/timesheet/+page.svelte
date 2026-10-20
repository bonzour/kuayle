<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { CalendarClock, RefreshCw } from 'lucide-svelte';
	import * as Select from '$lib/components/ui/select';
	import { Button } from '$lib/components/ui/button';
	import SidebarToggle from '$lib/components/layout/SidebarToggle.svelte';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import TimesheetFilter from '$lib/features/timesheets/TimesheetFilter.svelte';
	import TimesheetPeriodPicker from '$lib/features/timesheets/TimesheetPeriodPicker.svelte';
	import TimesheetGrid from '$lib/features/timesheets/TimesheetGrid.svelte';
	import TimesheetWorklogDetail from '$lib/features/timesheets/TimesheetWorklogDetail.svelte';
	import { personLabel } from '$lib/features/timesheets/people';
	import { periodForDate } from '$lib/features/timesheets/period';
	import { allTimesheetFilters, changeMode, changeWorkspaces, reconcileFilters } from '$lib/features/timesheets/selection';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getTimesheet, getTimesheetOptions } from '$lib/api/timesheets';
	import type { TimesheetFilters, TimesheetMode, TimesheetOptions, TimesheetSelection, TimesheetGrid as TimesheetGridData, TimesheetPeriod, TimesheetWorkspaceOption, TimesheetGroup, TimesheetQuery } from '$lib/types/timesheets';
	import { m } from '$lib/paraglide/messages.js';

	let mode = $state<TimesheetMode>('personal');
	let owners = $state<TimesheetSelection>(null);
	let filters = $state<TimesheetFilters>(allTimesheetFilters());
	let options = $state<TimesheetOptions | null>(null);
	let loading = $state(true);
	let failed = $state(false);
	let feedback = $state('');
	let generation = 0;
	let grid = $state<TimesheetGridData | null>(null);
	let period = $state<TimesheetPeriod | null>(null);
	let referenceId = $state('');
	let reference = $state<TimesheetWorkspaceOption | null>(null);
	let groupPage = $state(1);
	let detail = $state<{ group: TimesheetGroup; date: string; query: TimesheetQuery; returnFocus: HTMLElement | null } | null>(null);

	function saveHistory() {
		if (!period || !authState.user?.id) return;
		replaceState('', { ...page.state, timesheet: $state.snapshot({ userId: authState.user.id, slug: page.params.workspaceSlug ?? '', mode, owners, filters, period, referenceId, groupPage }) });
	}

	function readQuery(): TimesheetQuery {
		return { ...period!, mode, owners, filters, reference_workspace_id: referenceId, page: groupPage };
	}

	function selectPeriod(value: TimesheetPeriod) {
		period = value; groupPage = 1; void load();
	}

	function openCell(group: TimesheetGroup, date: string, ownerId?: string) {
		detail = { group, date, query: $state.snapshot({ ...readQuery(), owners: ownerId ? [ownerId] : owners }), returnFocus: document.activeElement as HTMLElement | null };
	}

	function openPerson(id: string) {
		owners = [id]; groupPage = 1; void load();
	}

	function selectOwners(value: TimesheetSelection) {
		owners = value; groupPage = 1; feedback = ''; void load();
	}

	function accessChanged() {
		detail = null; grid = null; options = null;
		feedback = m['timesheet.access_changed']();
		void load();
	}

	const modeLabels = $derived({
		personal: m['timesheet.personal'](), person: m['timesheet.by_person'](), project: m['timesheet.by_project'](),
		workspace: m['timesheet.by_workspace'](), team: m['timesheet.by_team']()
	});
	const workspaceName = (id: string) => {
		const w = options?.workspaces.find((w) => w.id === id);
		return w ? `${w.name} (${w.slug})` : '';
	};
	const workspaceOptions = $derived(options?.workspaces.map((w) => ({ id: w.id, name: workspaceName(w.id) })) ?? []);
	const selectedWorkspaceIds = $derived(filters.workspaces ?? options?.workspaces.map((w) => w.id) ?? []);
	const teamOptions = $derived(options?.teams.filter((t) => selectedWorkspaceIds.includes(t.workspace_id)).map((t) => ({ id: t.id, name: `${t.name} · ${workspaceName(t.workspace_id)}`, team: t })) ?? []);
	const projectOptions = $derived([
		{ id: 'none', name: m['timesheet.no_project']() },
		...(options?.projects.filter((p) => selectedWorkspaceIds.includes(p.workspace_id)).map((p) => ({ id: p.id, name: `${p.name} · ${workspaceName(p.workspace_id)}` })) ?? [])
	]);

	const peopleOptions = $derived.by(() => {
		const people = grid?.people ?? [];
		return [
			...people.map(person => ({ id: person.id, name: personLabel(person, m['timesheet.unavailable_user']()) })),
			...(owners ?? []).filter(id => !people.some(person => person.id === id)).map(id => ({ id, name: `${m['timesheet.unavailable_user']()} · ${id.slice(-8)}` }))
		];
	});

	async function load(requestMode = mode, requestFilters = filters, recover = true) {
		const id = ++generation;
		const userId = authState.user?.id;
		const current = () => id === generation && userId === authState.user?.id;
		loading = true; failed = false; grid = null; detail = null;
		try {
			const result = await getTimesheetOptions(requestMode, requestFilters);
			if (!current()) return;
			options = result;
			const personal = requestMode === 'personal' ? result : await getTimesheetOptions('personal', allTimesheetFilters());
			if (!current()) return;
			const nextReference = personal.workspaces.find(w => w.id === referenceId)
				?? (!referenceId ? personal.workspaces.find(w => w.slug === page.params.workspaceSlug) : undefined)
				?? personal.workspaces[0];
			if (referenceId && nextReference && referenceId !== nextReference.id) feedback = m['timesheet.reference_changed']();
			reference = nextReference ?? null;
			referenceId = nextReference?.id ?? '';
			if (nextReference) {
				period ??= periodForDate('week', nextReference.today);
				const response = await getTimesheet(readQuery());
				if (!current()) return;
				if (response.total_count > 0 && response.page > Math.ceil(response.total_count / response.per_page)) {
					groupPage = Math.ceil(response.total_count / response.per_page);
					await load(requestMode, requestFilters, false);
					return;
				}
				grid = response;
				reference = response.reference;
			}
			if (current()) saveHistory();
		} catch (error) {
			if (!current()) return;
			grid = null; options = null;
			const status = (error as { status?: number })?.status;
			if (recover && (status === 403 || status === 404)) {
				feedback = m['timesheet.access_changed']();
				try {
					// Discover current access without broadening the user's selection.
					let admitted = await getTimesheetOptions('personal', allTimesheetFilters());
					if (!current()) return;
					const nextMode = admitted.available_modes.includes(requestMode) ? requestMode : 'personal';
					if (nextMode !== 'personal') admitted = await getTimesheetOptions(nextMode, allTimesheetFilters());
					if (!current()) return;
					mode = nextMode;
					if (nextMode === 'personal') owners = null;
					filters = reconcileFilters(requestFilters, admitted);
					await load(nextMode, filters, false);
				} catch { if (current()) failed = true; }
			} else failed = true;
		} finally { if (current()) loading = false; }
	}

	function selectMode(value: string) {
		if (loading || !options || !options.available_modes.includes(value as TimesheetMode) || value === mode) return;
		filters = changeMode(filters, value as TimesheetMode, options);
		mode = value as TimesheetMode;
		if (mode === 'personal') owners = null;
		groupPage = 1;
		feedback = m['timesheet.scope_changed']();
		void load();
	}

	function selectWorkspaces(value: TimesheetSelection) {
		if (!options) return;
		const next = changeWorkspaces(filters, value, options);
		feedback = JSON.stringify([next.teams, next.projects]) !== JSON.stringify([filters.teams, filters.projects]) ? m['timesheet.children_removed']() : '';
		filters = next;
		groupPage = 1;
		void load();
	}

	function selectChild(key: 'teams' | 'projects', value: TimesheetSelection) {
		filters = { ...filters, [key]: value };
		groupPage = 1;
		feedback = '';
		void load();
	}

	$effect(() => {
		const userId = authState.user?.id;
		const slug = page.params.workspaceSlug;
		untrack(() => {
			generation++;
			mode = 'personal'; owners = null;
			filters = allTimesheetFilters();
			options = null; grid = null; detail = null; reference = null; referenceId = ''; period = null; groupPage = 1;
			const saved = page.state.timesheet;
			if (saved && saved.userId === userId && saved.slug === slug) {
				mode = saved.mode; owners = saved.owners ?? null; filters = saved.filters; referenceId = saved.referenceId; groupPage = saved.groupPage;
				try {
					const restored = periodForDate(saved.period.period, saved.period.from);
					if (restored.from === saved.period.from && restored.to === saved.period.to) period = restored;
				} catch { /* Invalid session state falls back to the reference's current week. */ }
			}
			feedback = '';
			if (userId && slug) void load();
			else loading = false;
		});
		return () => { generation++; };
	});
	onDestroy(() => { generation++; });
</script>

<div class="flex h-full flex-col bg-[var(--color-bg)]">
	<header class="flex h-[49px] shrink-0 items-center gap-2 border-b border-[var(--app-border)] px-4 sm:px-6">
		<SidebarToggle />
		<CalendarClock size={16} class="text-[var(--color-text-secondary)]" />
		<h1 class="text-sm font-medium text-[var(--color-text-primary)]">{m['timesheet.title']()}</h1>
	</header>
	<div class="shrink-0 border-b border-[var(--app-border)] bg-[var(--color-bg-secondary)]/35">
		<div class="mx-auto grid w-full max-w-[1440px] gap-3 px-4 py-3 sm:grid-cols-2 sm:px-6 {mode === 'personal' ? 'xl:grid-cols-4' : 'xl:grid-cols-5'}">
			<div class="min-w-0 space-y-1">
				<p class="text-xs text-[var(--color-text-secondary)]">{m['timesheet.mode']()}</p>
				<Select.Root type="single" value={mode} disabled={loading || !options} onValueChange={(value) => value && selectMode(value)}>
					<Select.Trigger size="sm" aria-label={m['timesheet.mode']()} class="w-full bg-[var(--color-bg)]">{modeLabels[mode]}</Select.Trigger>
					<Select.Content>
						{#each options?.available_modes ?? ['personal'] as value}
							<Select.Item value={value}>{modeLabels[value as TimesheetMode]}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<TimesheetFilter label={m['timesheet.workspaces']()} selection={filters.workspaces} options={workspaceOptions} disabled={!options} onchange={selectWorkspaces} />
			<TimesheetFilter label={m['timesheet.teams']()} selection={filters.teams} options={teamOptions} disabled={!options || selectedWorkspaceIds.length === 0} onchange={(value) => selectChild('teams', value)} />
			<TimesheetFilter label={m['timesheet.projects']()} selection={filters.projects} options={projectOptions} disabled={!options || selectedWorkspaceIds.length === 0} onchange={(value) => selectChild('projects', value)} />
			{#if mode !== 'personal'}
				<TimesheetFilter label={m['timesheet.people']()} selection={owners} options={peopleOptions} disabled={!grid || loading} onchange={selectOwners} />
			{/if}
		</div>
	</div>
	<main class="flex-1 overflow-y-auto">
		<div class="mx-auto w-full max-w-[1440px] space-y-5 px-4 py-6 sm:px-6">
			<div class="flex items-center justify-between gap-3">
				<p role="status" class="text-sm text-[var(--color-text-secondary)]">{feedback}</p>
				<Button variant="outline" size="sm" disabled={loading} onclick={() => load()}><RefreshCw size={14} />{m['timesheet.refresh']()}</Button>
			</div>
			{#if period && reference}
				<TimesheetPeriodPicker value={period} {reference} disabled={loading} onchange={selectPeriod} />
			{/if}
			{#if loading}
				<LoadingState />
			{:else if failed}
				<ErrorState message={m['timesheet.failed_load']()} onretry={() => load()} />
			{:else if options}
				{#if options.workspaces.length === 0}
					<p class="text-sm text-[var(--color-text-secondary)]">{m['timesheet.no_workspaces']()}</p>
				{:else if options.scope.workspace_ids.length === 0}
					<p class="text-sm text-[var(--color-text-secondary)]">{m['timesheet.no_workspace_selected']()}</p>
				{:else}
					<section aria-label={m['timesheet.effective_scope']()} class="space-y-2">
						<h2 class="text-sm font-medium text-[var(--color-text-primary)]">{m['timesheet.effective_scope']()}</h2>
						<ul class="space-y-2 text-sm text-[var(--color-text-secondary)]">
							{#each options.workspaces.filter((w) => options?.scope.workspace_ids.includes(w.id)) as workspace (workspace.id)}
								<li>{workspaceName(workspace.id)} · {workspace.timezone} · {workspace.can_record_work ? m['timesheet.recording_allowed']() : m['timesheet.read_only']()}</li>
							{/each}
						</ul>
					</section>
				{/if}
				{#if grid && options.scope.workspace_ids.length > 0}
					{#if grid.empty_reason}
						<p class="rounded-lg border border-[var(--app-border)] p-4 text-sm text-[var(--color-text-secondary)]">{grid.empty_reason === 'period' ? m['timesheet.no_work']() : m['timesheet.no_results']()}</p>
					{/if}
					<TimesheetGrid value={grid} oncell={openCell} onperson={openPerson} onpage={(value) => { groupPage = value; void load(); }} />
				{/if}
			{/if}
		</div>
	</main>
</div>

{#if detail}
	<TimesheetWorklogDetail {...detail} onclose={() => detail = null} onaccesschanged={accessChanged} />
{/if}
