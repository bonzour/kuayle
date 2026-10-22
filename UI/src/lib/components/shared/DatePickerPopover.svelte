<script lang="ts">
	import { Calendar } from '$lib/components/ui/calendar';
	import DatePopover from './DatePopover.svelte';
	import DueDatePickerPanel from './DueDatePickerPanel.svelte';
	import { parseCalendarDate, serializeCalendarDate, formatCalendarDate, calendarDateFromApi } from '$lib/utils/calendar-date';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime.js';
	let { value = null, onchange, placeholder, colorClass = '', dueDateMode = false, disabled = false, allowClear = true }: {
		value: string | null; onchange: (date: string | null) => void; placeholder?: string;
		colorClass?: string; dueDateMode?: boolean; disabled?: boolean; allowClear?: boolean;
	} = $props();
	let open = $state(false);
	const label = $derived(placeholder ?? m['controls.select_date']());
	const dateValue = $derived(calendarDateFromApi(value));
</script>

<div class="min-w-0">
	<DatePopover bind:open {label} {colorClass} text={dateValue ? formatCalendarDate(dateValue, getLocale()) : label} {disabled} onclear={allowClear && value ? () => onchange(null) : undefined}>
		{#if dueDateMode}
			<DueDatePickerPanel value={dateValue ?? null} {onchange} close={() => open = false} />
		{:else}
			<Calendar type="single" value={parseCalendarDate(dateValue)} placeholder={parseCalendarDate(dateValue)} onValueChange={(date) => { if (date) { onchange(serializeCalendarDate(date)); open = false; } }} />
		{/if}
	</DatePopover>
</div>
