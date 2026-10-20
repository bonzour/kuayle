<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { CalendarClock, RefreshCw } from 'lucide-svelte';
	import * as Tabs from '$lib/components/ui/tabs';
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
	import { readTimesheetLocation, writeTimesheetLocation, type TimesheetLocation } from '$lib/features/timesheets/navigation';
	import { allTimesheetFilters, changeMode, changeWorkspaces, reconcileFilters } from '$lib/features/timesheets/selection';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { getTimesheet, getTimesheetOptions } from '$lib/api/timesheets';
	import type { TimesheetMode, TimesheetOptions, TimesheetSelection, TimesheetGrid as TimesheetGridData, TimesheetPeriod, TimesheetWorkspaceOption, TimesheetGroup, TimesheetQuery } from '$lib/types/timesheets';
	import { m } from '$lib/paraglide/messages.js';

	let view = $state<TimesheetLocation>(readTimesheetLocation(new URLSearchParams()));
	const mode = $derived(view.mode);
	const owners = $derived(view.owners);
	const filters = $derived(view.filters);
	const period = $derived(view.period);
	let options = $state<TimesheetOptions | null>(null);
	let loading = $state(true);
	let failed = $state(false);
	let invalidLink = $state(false);
	let feedback = $state('');
	let generation = 0;
	let grid = $state<TimesheetGridData | null>(null);
	let reference = $state<TimesheetWorkspaceOption | null>(null);
	let heading: HTMLHeadingElement;
	let returnTabFocus = $state<HTMLElement | null>(null);
	let detail = $state<{ group: TimesheetGroup; date: string; query: TimesheetQuery; returnFocus: HTMLElement | null } | null>(null);
	let normalized: { href: string; userId: string | undefined; generation: number } | null = null;
	let pendingFeedback: { href: string; message: string } | null = null;
	let context = '';

	function readQuery(value = view): TimesheetQuery {
		return { ...value.period!, mode: value.mode, owners: value.owners, filters: value.filters, reference_workspace_id: value.referenceId, page: value.groupPage };
	}

	function navigate(value: TimesheetLocation, message = '', replaceState = false) {
		const url = writeTimesheetLocation(page.url, value);
		if (url.href === page.url.href) return;
		const active = document.activeElement as HTMLElement | null;
		if (active?.getAttribute('role') === 'tab') returnTabFocus = active;
		generation++;
		loading = true; grid = null; detail = null;
		pendingFeedback = { href: url.href, message };
		void goto(url, { replaceState, noScroll: true, keepFocus: true });
	}

	function selectPeriod(value: TimesheetPeriod) {
		navigate({ ...view, period: value, periodKind: value.period, groupPage: 1 });
	}

	function openCell(group: TimesheetGroup, date: string, ownerId?: string) {
		detail = { group, date, query: $state.snapshot({ ...readQuery(), owners: ownerId ? [ownerId] : owners }), returnFocus: document.activeElement as HTMLElement | null };
	}

	function selectOwners(value: TimesheetSelection) {
		navigate({ ...view, owners: value, groupPage: 1 });
	}

	function accessChanged() {
		detail = null;
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

	async function load(request = $state.snapshot(view), recover = true) {
		const id = ++generation;
		const userId = authState.user?.id;
		const url = new URL(page.url);
		const current = () => id === generation && userId === authState.user?.id;
		const previousScope = grid?.mode === request.mode ? grid.scope.workspace_ids : [];
		view = request;
		loading = true; failed = false; invalidLink = false; grid = null; detail = null;
		try {
			const result = await getTimesheetOptions(request.mode, request.filters);
			if (!current()) return;
			options = result;
			const personal = request.mode === 'personal' ? result : await getTimesheetOptions('personal', allTimesheetFilters());
			if (!current()) return;
			const nextReference = personal.workspaces.find(w => w.id === request.referenceId)
				?? (!request.referenceId ? personal.workspaces.find(w => w.slug === page.params.workspaceSlug) : undefined)
				?? personal.workspaces[0];
			if (request.referenceId && nextReference && request.referenceId !== nextReference.id) {
				feedback = [feedback, m['timesheet.reference_changed']()].filter(Boolean).join(' ');
			}
			reference = nextReference ?? null;
			const resolved = { ...request, referenceId: nextReference?.id ?? '', period: request.period ?? (nextReference ? periodForDate(request.periodKind, nextReference.today) : null) };
			if (nextReference) {
				const response = await getTimesheet(readQuery(resolved));
				if (!current()) return;
				const lastPage = Math.max(1, Math.ceil(response.total_count / response.per_page));
				if (response.page > lastPage) {
					await load({ ...resolved, groupPage: lastPage }, false);
					return;
				}
				grid = response;
				reference = response.reference;
			}
			view = resolved;
			const canonical = writeTimesheetLocation(url, resolved);
			if (canonical.href !== page.url.href) {
				normalized = { href: canonical.href, userId, generation: id };
				await goto(canonical, { replaceState: true, noScroll: true, keepFocus: true });
			}
		} catch (error) {
			if (!current()) return;
			grid = null; options = null; reference = null;
			const status = (error as { status?: number })?.status;
			if (recover && (status === 403 || status === 404)) {
				feedback = m['timesheet.access_changed']();
				try {
					let admitted = await getTimesheetOptions('personal', allTimesheetFilters());
					if (!current()) return;
					const nextMode = admitted.available_modes.includes(request.mode) ? request.mode : 'personal';
					if (nextMode !== 'personal') admitted = await getTimesheetOptions(nextMode, allTimesheetFilters());
					if (!current()) return;
					// A lost administrative mode must not turn All into all personal workspaces.
					const requestedFilters = nextMode !== request.mode && request.filters.workspaces === null
						? { ...request.filters, workspaces: previousScope } : request.filters;
					await load({ ...request, mode: nextMode, owners: nextMode === 'personal' ? null : request.owners, filters: reconcileFilters(requestedFilters, admitted) }, false);
				} catch { if (current()) failed = true; }
			} else {
				invalidLink = status === 400;
				failed = true;
			}
		} finally { if (current()) loading = false; }
	}

	function selectMode(value: string) {
		if (loading || !options || !options.available_modes.includes(value as TimesheetMode) || value === mode) return;
		navigate({ ...view, mode: value as TimesheetMode, owners: value === 'personal' ? null : owners, filters: changeMode(filters, value as TimesheetMode, options), groupPage: 1 }, m['timesheet.scope_changed']());
	}

	function selectWorkspaces(value: TimesheetSelection) {
		if (!options) return;
		const next = changeWorkspaces(filters, value, options);
		const message = JSON.stringify([next.teams, next.projects]) !== JSON.stringify([filters.teams, filters.projects]) ? m['timesheet.children_removed']() : '';
		navigate({ ...view, filters: next, groupPage: 1 }, message);
	}

	function selectChild(key: 'teams' | 'projects', value: TimesheetSelection) {
		navigate({ ...view, filters: { ...filters, [key]: value }, groupPage: 1 });
	}

	$effect(() => {
		if (!loading && returnTabFocus) {
			const target = returnTabFocus;
			returnTabFocus = null;
			if (document.activeElement === document.body) (target.isConnected ? target : heading)?.focus();
		}
	});

	$effect(() => {
		const userId = authState.user?.id;
		const slug = page.params.workspaceSlug;
		const href = page.url.href;
		untrack(() => {
			if (normalized?.href === href && normalized.userId === userId && normalized.generation === generation) {
				normalized = null;
				return;
			}
			generation++;
			grid = null; detail = null;
			if (context !== `${userId}/${slug}`) {
				options = null; reference = null;
				context = `${userId}/${slug}`;
			}
			feedback = pendingFeedback?.href === href ? pendingFeedback.message : '';
			pendingFeedback = null;
			try {
				const requested = readTimesheetLocation(new URL(href).searchParams);
				view = requested;
				if (userId && slug) void load(requested);
				else loading = false;
			} catch {
				options = null; reference = null;
				loading = false; failed = true; invalidLink = true;
			}
		});
	});
	onDestroy(() => { generation++; });
</script>

<div class="flex h-full min-w-0 flex-col bg-[var(--color-bg)]">
	<header class="flex h-[49px] shrink-0 items-center gap-2 border-b border-[var(--app-border)] px-4 sm:px-6">
		<SidebarToggle />
		<CalendarClock size={16} class="text-[var(--color-text-secondary)]" />
		<h1 bind:this={heading} tabindex="-1" class="text-sm font-medium text-[var(--color-text-primary)]">{m['timesheet.title']()}</h1>
	</header>
	<Tabs.Root value={options ? mode : 'personal'} onValueChange={selectMode} activationMode="manual" disabled={loading || !options || invalidLink} class="min-h-0 min-w-0 flex-1 gap-0">
		<div class="shrink-0 border-b border-[var(--app-border)] bg-[var(--color-bg-secondary)]/35">
			<div class="mx-auto w-full max-w-[1440px] space-y-3 px-4 py-3 sm:px-6">
				<div class="max-w-full overflow-x-auto">
					<Tabs.List aria-label={m['timesheet.mode']()} class="h-8 w-fit rounded-lg border border-[var(--app-border)] bg-[var(--color-bg)] p-0.5">
						{#each options?.available_modes ?? ['personal'] as value}
							<Tabs.Trigger value={value} class="h-6 shrink-0 rounded-md px-2.5 text-xs">{modeLabels[value as TimesheetMode]}</Tabs.Trigger>
						{/each}
					</Tabs.List>
				</div>
				<div class="grid gap-3 sm:grid-cols-2 {mode === 'personal' ? 'xl:grid-cols-3' : 'xl:grid-cols-4'}">
					<TimesheetFilter label={m['timesheet.workspaces']()} selection={filters.workspaces} options={workspaceOptions} disabled={!options} onchange={selectWorkspaces} />
					<TimesheetFilter label={m['timesheet.teams']()} selection={filters.teams} options={teamOptions} disabled={!options || selectedWorkspaceIds.length === 0} onchange={(value) => selectChild('teams', value)} />
					<TimesheetFilter label={m['timesheet.projects']()} selection={filters.projects} options={projectOptions} disabled={!options || selectedWorkspaceIds.length === 0} onchange={(value) => selectChild('projects', value)} />
					{#if mode !== 'personal'}
						<TimesheetFilter label={m['timesheet.people']()} selection={owners} options={peopleOptions} disabled={!grid || loading} onchange={selectOwners} />
					{/if}
				</div>
			</div>
		</div>
		<Tabs.Content value={options ? mode : 'personal'} class="min-h-0 min-w-0 flex-1 overflow-y-auto">
			<main>
				<div class="mx-auto w-full max-w-[1440px] space-y-5 px-4 py-6 sm:px-6">
					<div class="flex items-center justify-between gap-3">
						<p role="status" class="text-sm text-[var(--color-text-secondary)]">{feedback}</p>
						<Button variant="outline" size="sm" disabled={loading || invalidLink} onclick={() => load()}><RefreshCw size={14} />{m['timesheet.refresh']()}</Button>
					</div>
					{#if period && reference}
						<TimesheetPeriodPicker value={period} {reference} disabled={loading} onchange={selectPeriod} />
					{/if}
					{#if loading}
						<LoadingState />
					{:else if invalidLink}
						<ErrorState message={m['timesheet.invalid_url']()} retryLabel={m['timesheet.reset_view']()} onretry={() => navigate(readTimesheetLocation(new URLSearchParams()), '', true)} />
					{:else if failed}
						<ErrorState message={m['timesheet.failed_load']()} retryLabel={m['timesheet.retry']()} onretry={() => load()} />
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
							<TimesheetGrid value={grid} oncell={openCell} onperson={(id) => selectOwners([id])} onpage={(value) => navigate({ ...view, groupPage: value })} />
						{/if}
					{/if}
				</div>
			</main>
		</Tabs.Content>
	</Tabs.Root>
</div>

{#if detail}
	<TimesheetWorklogDetail {...detail} fallbackFocus={() => heading?.focus()} onclose={() => detail = null} onaccesschanged={accessChanged} />
{/if}
