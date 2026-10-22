<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	import { DateField } from 'bits-ui';
	import { Calendar } from '$lib/components/ui/calendar';
	import DatePopover from './DatePopover.svelte';
	import { parseCalendarDate, serializeCalendarDate, formatCalendarDate } from '$lib/utils/calendar-date';
	import { getLocale } from '$lib/paraglide/runtime.js';

	// ISO value/callback; localized editable segments. Incomplete input emits an empty draft.
	let { id, label, value, onchange, min, max, disabled = false, error = '', required = true }: {
		id: string; label: string; value: string; onchange: (value: string) => void;
		min?: string; max?: string; disabled?: boolean; error?: string; required?: boolean;
	} = $props();
	let open = $state(false);
	const calendarValue = $derived(parseCalendarDate(value));
</script>

<div class="min-w-0 space-y-1" data-date-field={id} data-value={value}>
	<DateField.Root value={calendarValue} onValueChange={(date) => onchange(date ? serializeCalendarDate(date) : '')} minValue={parseCalendarDate(min)} maxValue={parseCalendarDate(max)} {disabled} {required} locale={getLocale()} errorMessageId={error ? `${id}-error` : undefined}>
		<DateField.Label class="text-xs font-medium">{label}</DateField.Label>
		<div class="flex min-w-0 items-center gap-1">
			<DateField.Input {id} aria-label={label} aria-invalid={!!error} aria-describedby={error ? `${id}-error` : undefined} class="flex min-h-9 min-w-0 flex-1 items-center rounded-lg border border-input bg-[var(--color-bg)] px-2 text-sm focus-within:ring-2 focus-within:ring-ring aria-invalid:border-destructive data-disabled:opacity-50">
				{#snippet children({ segments })}
					{#each segments as segment}<DateField.Segment part={segment.part} class="rounded px-0.5 tabular-nums focus:bg-accent focus:outline-none data-placeholder:text-muted-foreground">
							{#snippet child({ props })}<span {...props} aria-label={segment.part === 'day' ? m['controls.date_day']() : segment.part === 'month' ? m['controls.date_month']() : segment.part === 'year' ? m['controls.date_year']() : undefined} aria-describedby={error ? `${id}-error` : undefined}>{segment.value}</span>{/snippet}
						</DateField.Segment>{/each}
				{/snippet}
			</DateField.Input>
			<DatePopover bind:open {label} text={formatCalendarDate(value, getLocale())} {disabled} iconOnly onclear={!required && value ? () => onchange('') : undefined}>
				<Calendar {disabled} type="single" value={calendarValue} placeholder={calendarValue ?? parseCalendarDate(max)} minValue={parseCalendarDate(min)} maxValue={parseCalendarDate(max)} onValueChange={(date) => { if (date) { onchange(serializeCalendarDate(date)); open = false; } }} />
			</DatePopover>
		</div>
	</DateField.Root>
	{#if error}<p id={`${id}-error`} role="alert" class="text-xs text-destructive">{error}</p>{/if}
</div>
