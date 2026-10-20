<script lang="ts">
	import { ChevronDown, ChevronRight } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import TeamIcon from '$lib/components/shared/TeamIcon.svelte';
	import type { TimesheetGrid, TimesheetGroup } from '$lib/types/timesheets';
	import { formatTimesheetDate, formatTimesheetMinutes } from './period';
	import { personLabel } from './people';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let { value, oncell, onpage, onperson }: { value: TimesheetGrid; oncell: (group: TimesheetGroup, date: string, ownerId?: string) => void; onperson: (id: string) => void; onpage: (page: number) => void } = $props();
	let collapsed = $state(new Set<string>());
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

<div class="space-y-3">
	<p class="text-xs text-[var(--color-text-secondary)]">{m['timesheet.totals_note']()} {m['timesheet.classification_note']()}</p>
	<!-- svelte-ignore a11y_no_noninteractive_tabindex (Keyboard users need to scroll the month grid.) -->
	<div role="region" aria-label={m['timesheet.grid']()} tabindex="0" class="max-w-full overflow-auto rounded-lg border border-[var(--app-border)] focus-visible:outline-2 focus-visible:outline-ring">
		<table class="w-full border-collapse text-sm tabular-nums">
			<caption class="sr-only">{m['timesheet.grid']()}</caption>
			<thead>
				<tr class="border-b border-[var(--app-border)] bg-[var(--color-bg-secondary)]">
					<th scope="col" class="sticky left-0 z-10 min-w-[280px] bg-[var(--color-bg-secondary)] px-3 py-3 text-left font-medium">{m['timesheet.work']()}</th>
					{#each value.days as day}
						<th scope="col" aria-current={day === value.reference.today ? 'date' : undefined} class="min-w-[88px] px-3 py-3 text-center font-medium" class:bg-[var(--color-bg-tertiary)]={new Date(`${day}T00:00:00Z`).getUTCDay() % 6 === 0}>
							<span class="block text-xs text-[var(--color-text-secondary)]">{formatTimesheetDate(day, getLocale(), { weekday: 'short', day: undefined, month: undefined })}</span>
							{formatTimesheetDate(day, getLocale())}
						</th>
					{/each}
					<th scope="col" class="min-w-[95px] px-3 py-3 text-right font-medium">{m['timesheet.total']()}</th>
				</tr>
			</thead>
			<tbody>
				{#each rows as { group, depth, ownerId } (group.key)}
					<tr class="border-b border-[var(--app-border)] last:border-0" class:font-medium={group.type !== 'issue'}>
						<th scope="row" class="sticky left-0 z-10 max-w-[380px] bg-[var(--color-bg)] py-2 pr-3 text-left font-normal" class:font-medium={group.type !== 'issue'} style:padding-left={`${12 + depth * 16}px`}>
							<div class="flex items-center gap-1.5">
								{#if group.children.length}
									<button class="rounded p-1 focus-visible:outline-2 focus-visible:outline-ring" aria-expanded={!collapsed.has(group.key)} aria-label={label(group)} onclick={() => toggle(group.key)}>{#if collapsed.has(group.key)}<ChevronRight size={14} />{:else}<ChevronDown size={14} />{/if}</button>
								{/if}
								{#if group.team}<TeamIcon team={group.team} size={14} />{/if}
								{#if group.type === 'issue'}
									<a href={`/${group.workspace_slug}/issue/${group.identifier}`} class="min-w-0 rounded hover:underline focus-visible:outline-2 focus-visible:outline-ring"><span class="text-[var(--color-text-secondary)]">{group.identifier}</span>{' '}{group.name}</a>
								{:else}<span class="truncate" title={label(group)}>{label(group)}{#if depth === 0 && group.workspace_slug && group.type !== 'workspace'}<span class="text-xs text-[var(--color-text-secondary)]"> · {group.workspace_slug}</span>{/if}</span>{/if}
								{#if value.mode === 'person' && group.person}
									<button class="shrink-0 rounded px-1 text-xs text-[var(--color-text-secondary)] hover:underline focus-visible:outline-2 focus-visible:outline-ring" aria-label={`${m['timesheet.open_person']()} · ${label(group)}`} onclick={() => onperson(group.person!.id)}>{m['timesheet.open_person']()}</button>
								{/if}
							</div>
						</th>
						{#each group.daily_totals as day}
							<td class="px-2 py-2 text-center">
								{#if group.type === 'issue' && day.minutes > 0}
									<button class="w-full rounded px-1 py-1 hover:bg-[var(--color-bg-secondary)] focus-visible:outline-2 focus-visible:outline-ring" aria-label={`${group.identifier} · ${day.date} · ${formatTimesheetMinutes(day.minutes)}`} onclick={() => oncell(group, day.date, ownerId)}>{formatTimesheetMinutes(day.minutes)}</button>
								{:else}<span class:text-[var(--color-text-tertiary)]={day.minutes === 0}>{day.minutes ? formatTimesheetMinutes(day.minutes) : '–'}</span>{/if}
							</td>
						{/each}
						<td class="px-3 py-2 text-right">{formatTimesheetMinutes(group.total_minutes)}</td>
					</tr>
				{/each}
			</tbody>
			<tfoot class="border-t border-[var(--app-border)] bg-[var(--color-bg-secondary)] font-semibold">
				<tr><th scope="row" class="sticky left-0 bg-[var(--color-bg-secondary)] px-3 py-3 text-left">{m['timesheet.total']()}</th>{#each value.daily_totals as day}<td class="px-2 py-3 text-center">{formatTimesheetMinutes(day.minutes)}</td>{/each}<td class="px-3 py-3 text-right">{formatTimesheetMinutes(value.total_minutes)}</td></tr>
			</tfoot>
		</table>
	</div>
	{#if value.total_count > value.per_page}
		<div class="flex items-center justify-end gap-3" aria-label={m['timesheet.group_pages']()}>
			<span class="text-xs text-[var(--color-text-secondary)]">{m['timesheet.page']({ page: value.page, pages: Math.ceil(value.total_count / value.per_page) })}</span>
			<Button variant="outline" size="sm" disabled={value.page <= 1} onclick={() => onpage(value.page - 1)}>{m['timesheet.previous']()}</Button>
			<Button variant="outline" size="sm" disabled={!value.has_more} onclick={() => onpage(value.page + 1)}>{m['timesheet.next']()}</Button>
		</div>
	{/if}
</div>
