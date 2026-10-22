<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { retainedGroups, type TimesheetPresentation } from './presentation';
	import ResultPagination from '$lib/components/shared/ResultPagination.svelte';
	import { ChevronDown, ChevronRight, Plus } from 'lucide-svelte';
	import TeamIcon from '$lib/components/shared/TeamIcon.svelte';
	import type { TimesheetGrid, TimesheetGroup } from '$lib/types/timesheets';
	import { formatTimesheetDate, formatTimesheetHours, formatTimesheetDays } from './period';
	import { personLabel } from './people';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let { value, initialPresentation, oncell, oncreate, onpage, onperson }: { value: TimesheetGrid; initialPresentation?: TimesheetPresentation; oncell: (group: TimesheetGroup, date: string, ownerId?: string) => void; oncreate: (group: TimesheetGroup, date: string, ownerId?: string) => void; onperson: (id: string) => void; onpage: (page: number) => void } = $props();
	let collapsed = $state(new Set(untrack(() => retainedGroups(initialPresentation?.collapsed ?? [], value.groups))));
	let viewport: HTMLDivElement;
	let root: HTMLDivElement;
	export function capture(): TimesheetPresentation {
		const active = document.activeElement as HTMLElement | null;
		return { collapsed: [...collapsed], scrollTop: viewport.scrollTop, scrollLeft: viewport.scrollLeft,
			pageScroll: root.closest('[data-slot="tabs-content"]')?.scrollTop ?? 0,
			height: root.getBoundingClientRect().height,
			focus: active && root.contains(active) ? active.dataset.focusKey ?? null : null };
	}
	onMount(() => {
		if (!initialPresentation) return;
		const frame = requestAnimationFrame(() => {
			viewport.scrollTop = initialPresentation!.scrollTop; viewport.scrollLeft = initialPresentation!.scrollLeft;
			const parent = root.closest('[data-slot="tabs-content"]');
			if (parent) parent.scrollTop = initialPresentation!.pageScroll;
			if (initialPresentation!.focus && document.activeElement === document.body) {
				const target = [...root.querySelectorAll<HTMLElement>('[data-focus-key]')].find(node => node.dataset.focusKey === initialPresentation!.focus);
				(target ?? viewport).focus({ preventScroll: true });
			}
		});
		return () => cancelAnimationFrame(frame);
	});
	const label = (group: TimesheetGroup) => group.person ? personLabel(group.person, m['timesheet.unavailable_user']()) : group.type === 'no_project' ? m['timesheet.no_project']() : group.type === 'unavailable_project' ? m['timesheet.unavailable_project']() : group.name;
	const rows = $derived.by(() => {
		const result: { group: TimesheetGroup; depth: number; ownerId?: string }[] = [];
		function visit(groups: TimesheetGroup[], depth: number, ownerId?: string) {
			for (const group of groups) {
				const rowOwner = group.person?.id ?? ownerId;
				result.push({ group, depth, ownerId: rowOwner });
				if (!collapsed.has(group.key)) visit(group.children, depth + 1, rowOwner);
			}
		}
		visit(value.groups, 0);
		return result;
	});
	function toggle(key: string) {
		const next = new Set(collapsed);
		if (next.has(key)) next.delete(key); else next.add(key);
		collapsed = next;
	}
</script>

<div bind:this={root} class="space-y-3">
	<p class="text-xs text-[var(--color-text-secondary)]">{m['timesheet.classification_note']()} {m['controls.scroll_hint']()}</p>
	<!-- svelte-ignore a11y_no_noninteractive_tabindex (Keyboard users need to scroll the month grid.) -->
	<div bind:this={viewport} data-focus-key="viewport" role="region" aria-label={m['timesheet.grid']()} tabindex="0" class="max-h-[65dvh] max-w-full overflow-auto rounded-lg border border-[var(--app-border)] focus-visible:outline-2 focus-visible:outline-ring">
		<table class="w-full table-fixed border-separate border-spacing-0 text-sm tabular-nums">
			<caption class="sr-only">{m['timesheet.grid']()}</caption>
			<thead class="sticky top-0 z-20">
				<tr class="border-b border-[var(--app-border)] bg-[var(--color-bg-secondary)]">
					<th scope="col" class="sticky left-0 z-10 w-36 min-w-36 sm:w-[220px] sm:min-w-[220px] lg:w-[280px] lg:min-w-[280px] bg-[var(--color-bg-secondary)] px-3 py-3 text-left font-medium">{m['timesheet.work']()}</th>
					{#each value.days as day}
						<th scope="col" aria-current={day === value.reference.today ? 'date' : undefined} class="w-[88px] min-w-[88px] bg-[var(--color-bg-secondary)] px-3 py-3 text-center font-medium" class:bg-[var(--color-bg-tertiary)]={new Date(`${day}T00:00:00Z`).getUTCDay() % 6 === 0}>
							<span class="block text-xs text-[var(--color-text-secondary)]">{formatTimesheetDate(day, getLocale(), { weekday: 'short', day: undefined, month: undefined })}</span>
							{formatTimesheetDate(day, getLocale())}
							{#if day === value.reference.today}<span class="block text-[10px]">{m['timesheet.today']()}</span>{/if}
							{#if new Date(`${day}T00:00:00Z`).getUTCDay() % 6 === 0}<span class="sr-only">{m['controls.weekend']()}</span>{/if}
						</th>
					{/each}
					<th scope="col" class="w-24 min-w-24 lg:sticky lg:right-0 bg-[var(--color-bg-secondary)] px-3 py-3 text-right font-medium">{m['timesheet.total']()}</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as { group, depth, ownerId } (group.key)}
					<tr class="border-b border-[var(--app-border)] last:border-0" class:font-medium={group.type !== 'issue'}>
						<th scope="row" class="row-label sticky left-0 z-10 bg-[var(--color-bg)] py-2 pr-3 text-left font-normal" class:font-medium={group.type !== 'issue'} style:--depth={depth}>
							<div class="flex items-center gap-1.5">
								{#if group.children.length}
									<button data-focus-key={JSON.stringify([group.key, 'toggle'])} class="min-h-8 min-w-8 shrink-0 rounded p-1 focus-visible:outline-2 focus-visible:outline-ring" aria-expanded={!collapsed.has(group.key)} aria-label={label(group)} onclick={() => toggle(group.key)}>{#if collapsed.has(group.key)}<ChevronRight size={14} />{:else}<ChevronDown size={14} />{/if}</button>
								{/if}
								{#if group.team}<TeamIcon team={group.team} size={14} />{/if}
								{#if group.type === 'issue'}
									<a href={`/${group.workspace_slug}/issue/${group.identifier}`} data-focus-key={JSON.stringify([group.key, 'issue'])} class="min-w-0 break-words rounded hover:underline focus-visible:outline-2 focus-visible:outline-ring"><span class="text-[var(--color-text-secondary)]">{group.identifier}</span>{' '}{group.name}</a>
								{:else}<span class="min-w-0 break-words">{label(group)}{#if depth === 0 && group.workspace_slug && group.type !== 'workspace'}<span class="text-xs text-[var(--color-text-secondary)]"> · {group.workspace_slug}</span>{/if}</span>{/if}
								{#if value.mode === 'person' && group.person}
									<button class="min-h-8 shrink-0 rounded px-1 text-xs text-[var(--color-text-secondary)] hover:underline focus-visible:outline-2 focus-visible:outline-ring" aria-label={`${m['timesheet.open_person']()} · ${label(group)}`} onclick={() => onperson(group.person!.id)}>{m['timesheet.open_person']()}</button>
								{/if}
							</div>
						</th>
						{#each group.daily_totals as day}
							<td class="min-w-[88px] px-2 py-2 text-center">
								{#if group.type === 'issue' && day.minutes > 0}
									<button data-focus-key={JSON.stringify([group.key, day.date])} class="min-h-8 w-full rounded px-1 py-1 underline decoration-dotted underline-offset-4 hover:bg-[var(--color-bg-secondary)] focus-visible:outline-2 focus-visible:outline-ring" aria-label={`${group.identifier} · ${day.date} · ${formatTimesheetHours(day.minutes, getLocale())}`} onclick={() => oncell(group, day.date, ownerId)}>{formatTimesheetHours(day.minutes, getLocale())}</button>
								{:else if group.type === 'issue' && value.workspaces.some(w => w.id === group.workspace_id && w.can_record_work && day.date <= w.today)}
									<button data-focus-key={JSON.stringify([group.key, day.date])} class="min-h-8 w-full rounded px-1 py-1 text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-secondary)] focus-visible:outline-2 focus-visible:outline-ring" aria-label={`${m['efforts.worklogs.create']()} · ${group.identifier} · ${day.date}`} onclick={() => oncreate(group, day.date, ownerId)}><Plus size={16} class="mx-auto" /></button>
								{:else}<span class:text-[var(--color-text-tertiary)]={day.minutes === 0}>{day.minutes ? formatTimesheetDays(day.minutes, getLocale()) : '–'}</span>{/if}
							</td>
						{/each}
						<td class="bg-[var(--color-bg)] px-3 py-2 text-right lg:sticky lg:right-0 lg:z-10">{formatTimesheetDays(group.total_minutes, getLocale())}</td>
					</tr>
				{/each}
			</tbody>
			<tfoot class="border-t border-[var(--app-border)] bg-[var(--color-bg-secondary)] font-semibold">
				<tr><th scope="row" class="sticky left-0 bg-[var(--color-bg-secondary)] px-3 py-3 text-left">{m['timesheet.total']()}</th>{#each value.daily_totals as day}<td class="px-2 py-3 text-center">{formatTimesheetDays(day.minutes, getLocale())}</td>{/each}<td class="bg-[var(--color-bg-secondary)] px-3 py-3 text-right lg:sticky lg:right-0">{formatTimesheetDays(value.total_minutes, getLocale())}</td></tr>
			</tfoot>
		</table>
	</div>
	<p class="text-xs text-muted-foreground">{m['controls.readonly_hint']()}</p>
	<ResultPagination page={value.page} perPage={value.per_page} total={value.total_count} label={m['timesheet.group_pages']()} onchange={onpage} />
</div>

<style>
	.row-label { padding-left: calc(8px + min(var(--depth), 2) * 8px); }
	@media (min-width: 640px) { .row-label { padding-left: calc(12px + var(--depth) * 16px); } }
</style>
