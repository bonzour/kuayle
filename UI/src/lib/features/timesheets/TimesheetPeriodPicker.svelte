<script lang="ts">
	import { parseDate } from '@internationalized/date';
	import { ChevronLeft, ChevronRight, CalendarDays } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import { Calendar } from '$lib/components/ui/calendar';
	import * as Popover from '$lib/components/ui/popover';
	import type { TimesheetPeriod, TimesheetWorkspaceOption } from '$lib/types/timesheets';
	import { formatTimesheetDate, movePeriod, periodForDate } from './period';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let { value, reference, disabled = false, onchange }: {
		value: TimesheetPeriod; reference: TimesheetWorkspaceOption; disabled?: boolean; onchange: (value: TimesheetPeriod) => void;
	} = $props();
	let open = $state(false);
	let calendar = $state<HTMLDivElement | null>(null);
	let trigger = $state<HTMLButtonElement | null>(null);
	let returnFocus = $state(false);
	$effect(() => {
		if (returnFocus && !open && !disabled) {
			returnFocus = false;
			trigger?.focus();
		}
	});
</script>

<div class="space-y-2">
	<div class="flex flex-wrap items-center gap-2" role="group" aria-label={m['timesheet.period']()}>
		<Button variant={value.period === 'week' ? 'secondary' : 'ghost'} size="sm" aria-pressed={value.period === 'week'} {disabled} onclick={() => onchange(periodForDate('week', value.from))}>{m['timesheet.week']()}</Button>
		<Button variant={value.period === 'month' ? 'secondary' : 'ghost'} size="sm" aria-pressed={value.period === 'month'} {disabled} onclick={() => onchange(periodForDate('month', value.from))}>{m['timesheet.month']()}</Button>
		<Button variant="outline" size="icon-sm" aria-label={m['timesheet.previous_period']()} {disabled} onclick={() => onchange(movePeriod(value, -1))}><ChevronLeft size={16} /></Button>
		<Popover.Root bind:open>
			<Popover.Trigger>
				{#snippet child({ props })}
					<Button {...props} bind:ref={trigger} variant="outline" size="sm" {disabled} aria-label={m['timesheet.select_period']()}><CalendarDays size={16} />{formatTimesheetDate(value.from, getLocale(), { year: 'numeric' })} – {formatTimesheetDate(value.to, getLocale(), { year: 'numeric' })}</Button>
				{/snippet}
			</Popover.Trigger>
			<Popover.Content align="start" class="w-auto p-0" onOpenAutoFocus={(event) => {
				const day = calendar?.querySelector<HTMLElement>('[data-bits-day][data-focused]');
				if (day) { event.preventDefault(); day.focus(); }
			}} onCloseAutoFocus={(event) => { if (returnFocus) event.preventDefault(); }}>
				<Calendar bind:ref={calendar} type="single" value={parseDate(value.from)} placeholder={parseDate(value.from)} weekStartsOn={1} onValueChange={(day) => { if (day) { returnFocus = true; onchange(periodForDate(value.period, day.toString())); open = false; } }} />
			</Popover.Content>
		</Popover.Root>
		<Button variant="outline" size="icon-sm" aria-label={m['timesheet.next_period']()} {disabled} onclick={() => onchange(movePeriod(value, 1))}><ChevronRight size={16} /></Button>
		<Button variant="outline" size="sm" {disabled} onclick={() => onchange(periodForDate(value.period, reference.today))}>{m['timesheet.current_period']()}</Button>
	</div>
	<p class="text-xs text-[var(--color-text-secondary)]">{m['timesheet.reference']()}: {reference.name} ({reference.slug}) · {reference.timezone} · {m['timesheet.today']()}: {reference.today}</p>
</div>
