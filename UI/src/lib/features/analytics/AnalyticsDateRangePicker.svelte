<script lang="ts">
	import { today, getLocalTimeZone } from '@internationalized/date';
	import { Button } from '$lib/components/ui/button';
	import DateRangePickerPopover from '$lib/components/shared/DateRangePickerPopover.svelte';
	import { m } from '$lib/paraglide/messages.js';
	let { startDate = '', endDate = '', onchange, allowClear = false }: {
		startDate: string; endDate: string; onchange: (start: string, end: string) => void; allowClear?: boolean;
	} = $props();
	function preset(days: number, select: (start: string, end: string) => void) {
		const end = today(getLocalTimeZone());
		select(end.subtract({ days: days - 1 }).toString(), end.toString());
	}
</script>

<DateRangePickerPopover {startDate} {endDate} {onchange} placeholder={m['insights.date_range']()} onclear={allowClear ? () => onchange('', '') : undefined}>
	{#snippet presets(select)}
		<div class="flex flex-wrap gap-1 border-b border-[var(--app-border)] p-2">
			<Button variant="ghost" size="sm" onclick={() => preset(30, select)}>{m['insights.days_30']()}</Button>
			<Button variant="ghost" size="sm" onclick={() => preset(90, select)}>{m['insights.days_90']()}</Button>
			<Button variant="ghost" size="sm" onclick={() => preset(180, select)}>{m['insights.months_6']()}</Button>
		</div>
	{/snippet}
</DateRangePickerPopover>
