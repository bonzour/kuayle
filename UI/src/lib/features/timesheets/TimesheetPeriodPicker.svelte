<script lang="ts">
	import { parseDate } from '@internationalized/date';
	import { ChevronLeft, ChevronRight } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import * as CalendarUI from '$lib/components/ui/calendar';
	import DatePopover from '$lib/components/shared/DatePopover.svelte';
	import type { TimesheetPeriod, TimesheetWorkspaceOption } from '$lib/types/timesheets';
	import { formatTimesheetDate, movePeriod, periodForDate } from './period';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';

	let { value, reference, disabled = false, onchange }: {
		value: TimesheetPeriod; reference: TimesheetWorkspaceOption; disabled?: boolean; onchange: (value: TimesheetPeriod) => void;
	} = $props();
	let open = $state(false);
	const displayPeriod = $derived(value.period === 'month'
		? formatTimesheetDate(value.from, getLocale(), { month: 'long', year: 'numeric', day: undefined })
		: new Intl.DateTimeFormat(getLocale(), { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' }).formatRange(parseDate(value.from).toDate('UTC'), parseDate(value.to).toDate('UTC')));
</script>

<div class="space-y-2">
	<div class="flex flex-wrap items-center gap-2" role="group" aria-label={m['timesheet.period']()}>
		<Button variant={value.period === 'week' ? 'secondary' : 'ghost'} size="sm" aria-pressed={value.period === 'week'} {disabled} onclick={() => onchange(periodForDate('week', value.from))}>{m['timesheet.week']()}</Button>
		<Button variant={value.period === 'month' ? 'secondary' : 'ghost'} size="sm" aria-pressed={value.period === 'month'} {disabled} onclick={() => onchange(periodForDate('month', value.from))}>{m['timesheet.month']()}</Button>
		<Button variant="outline" size="icon-sm" aria-label={m['timesheet.previous_period']()} {disabled} onclick={() => onchange(movePeriod(value, -1))}><ChevronLeft size={16} /></Button>
		<DatePopover bind:open label={m['timesheet.select_period']()} text={displayPeriod} {disabled}>
			<p class="max-w-72 px-3 pt-3 text-sm">{m['controls.period_selected']({ period: displayPeriod })}</p>
			<CalendarUI.Calendar type="single" value={parseDate(value.from)} placeholder={parseDate(value.from)} weekStartsOn={1} onValueChange={(day) => { if (day) { onchange(periodForDate(value.period, day.toString())); open = false; } }}>
				{#snippet day({ day })}
					<CalendarUI.Day data-period-selected={day.toString() >= value.from && day.toString() <= value.to ? '' : undefined} class={day.toString() >= value.from && day.toString() <= value.to ? 'bg-primary/15 border-y-2 border-primary rounded-none' : ''}>
						{#snippet child({ props })}
							<div {...props} data-today={day.toString() === reference.today ? '' : undefined} aria-current={day.toString() === reference.today ? 'date' : undefined}>{day.day}</div>
						{/snippet}
					</CalendarUI.Day>
				{/snippet}
			</CalendarUI.Calendar>
		</DatePopover>
		<Button variant="outline" size="icon-sm" aria-label={m['timesheet.next_period']()} {disabled} onclick={() => onchange(movePeriod(value, 1))}><ChevronRight size={16} /></Button>
		<Button variant="outline" size="sm" {disabled} onclick={() => onchange(periodForDate(value.period, reference.today))}>{m['timesheet.current_period']()}</Button>
	</div>
	<p class="text-xs text-[var(--color-text-secondary)]">{m['timesheet.reference']()}: {reference.name} ({reference.slug}) · {reference.timezone} · {m['timesheet.today']()}: {formatTimesheetDate(reference.today, getLocale(), { year: 'numeric' })}</p>
</div>
