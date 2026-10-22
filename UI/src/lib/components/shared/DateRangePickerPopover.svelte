<script lang="ts">
	import { MediaQuery } from 'svelte/reactivity';
	import type { Snippet } from 'svelte';
	import type { DateValue } from '@internationalized/date';
	import { RangeCalendar } from '$lib/components/ui/range-calendar';
	import DatePopover from './DatePopover.svelte';
	import { parseCalendarDate, serializeCalendarDate, formatCalendarDate, calendarDateFromApi } from '$lib/utils/calendar-date';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';
	let { startDate = null, endDate = null, onchange, onclear, isDateDisabled, numberOfMonths = 2, placeholder, disabled = false, presets }: {
		startDate: string | null; endDate: string | null; onchange: (start: string, end: string) => void;
		onclear?: () => void; isDateDisabled?: (date: DateValue) => boolean; numberOfMonths?: number;
		placeholder?: string; disabled?: boolean; presets?: Snippet<[(start: string, end: string) => void]>;
	} = $props();
	let open = $state(false);
	const wide = new MediaQuery('(min-width: 640px)');
	const label = $derived(placeholder ?? m['controls.select_range']());
	const start = $derived(calendarDateFromApi(startDate));
	const end = $derived(calendarDateFromApi(endDate));
	const calendarValue = $derived({ start: parseCalendarDate(start), end: parseCalendarDate(end) });
	const text = $derived(start && end ? `${formatCalendarDate(start, getLocale())} – ${formatCalendarDate(end, getLocale())}` : start ? formatCalendarDate(start, getLocale()) : label);
	function select(start: string, end: string) { onchange(start, end); open = false; }
</script>

<DatePopover bind:open {label} {text} {disabled} onclear={onclear && (startDate || endDate) ? onclear : undefined}>
	{@render presets?.(select)}
	{#key `${startDate}-${endDate}`}
		<RangeCalendar value={calendarValue} placeholder={calendarValue.start} onValueChange={(range) => { if (range?.start && range.end) select(serializeCalendarDate(range.start), serializeCalendarDate(range.end)); }} numberOfMonths={wide.current ? numberOfMonths : 1} {isDateDisabled} />
	{/key}
</DatePopover>
