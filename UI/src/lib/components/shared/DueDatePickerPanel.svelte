<script lang="ts">
	import { Calendar } from '$lib/components/ui/calendar';
	import { preferencesState } from '$lib/features/preferences/preferences.state.svelte';
	import { parseCalendarDate, serializeCalendarDate, formatCalendarDate, calendarDateFromApi } from '$lib/utils/calendar-date';
	import { today, getLocalTimeZone, type DateValue } from '@internationalized/date';
	import { CalendarDays, X } from 'lucide-svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { getLocale, setLocale } from '$lib/paraglide/runtime.js';

	let {
		value = null,
		onchange,
		close,
		clearLabel
	}: {
		value: string | null;
		onchange: (date: string | null) => void | Promise<void>;
		close?: () => void;
		clearLabel?: string;
	} = $props();

	const effectiveClearLabel = $derived(clearLabel ?? m['sharedComponents.due_date_picker.no_due_date']());

	const presets = $derived([
		{ label: m['sharedComponents.due_date_picker.today'](), value: today(getLocalTimeZone()).toString() },
		{ label: m['sharedComponents.due_date_picker.tomorrow'](), value: today(getLocalTimeZone()).add({ days: 1 }).toString() },
		{ label: m['sharedComponents.due_date_picker.next_week'](), value: today(getLocalTimeZone()).add({ days: 7 }).toString() },
	]);
	const recentDueDates = $derived(preferencesState.recentDueDates.filter((date) => !presets.some((preset) => preset.value === date)));
	const calendarValue = $derived(parseCalendarDate(calendarDateFromApi(value)));

	async function selectDate(date: string | null) {
		await onchange(date);
		if (date) preferencesState.addRecentDueDate(date);
		close?.();
	}

	function handleCalendarSelect(dateValue: DateValue | undefined) {
		if (!dateValue) return;
		void selectDate(serializeCalendarDate(dateValue));
	}

	function formatDisplayDate(date: string) { return formatCalendarDate(date, getLocale()); }
</script>

<div class="mx-auto grid w-fit gap-2 p-2 sm:grid-cols-[10.75rem_max-content]">
	<div class="min-w-0 space-y-2">
		{#if recentDueDates.length > 0}
			<div>
				<div class="px-2 pb-1 text-[10px] font-medium uppercase text-[var(--color-text-tertiary)]">{m['sharedComponents.due_date_picker.recent']()}</div>
				<div class="space-y-0.5">
					{#each recentDueDates as date (date)}
						<button
							type="button"
							class="flex min-h-8 w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)]"
							onclick={() => selectDate(date)}
						>
							<CalendarDays size={13} />
							<span class="truncate">{formatDisplayDate(date)}</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<div>
			<div class="px-2 pb-1 text-[10px] font-medium uppercase text-[var(--color-text-tertiary)]">{m['sharedComponents.due_date_picker.presets']()}</div>
			<div class="space-y-0.5">
				{#each presets as preset (preset.label)}
					<button
						type="button"
						class="flex min-h-8 w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-xs text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)]"
						onclick={() => selectDate(preset.value)}
					>
						<span>{preset.label}</span>
						<span class="text-[10px] text-[var(--color-text-tertiary)]">{formatDisplayDate(preset.value)}</span>
					</button>
				{/each}

				<button
					type="button"
					class="flex min-h-8 w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-[var(--color-text-tertiary)] hover:bg-[var(--color-bg-hover)] hover:text-[var(--color-text-primary)]"
					onclick={() => selectDate(null)}
				>
					<X size={13} />
					{effectiveClearLabel}
				</button>
			</div>
		</div>
	</div>

	<div class="w-max overflow-hidden rounded-xl border border-[var(--app-border)] bg-[var(--color-bg)]/40">
		{#key value}
			<Calendar
				type="single"
				value={calendarValue}
				onValueChange={handleCalendarSelect}
			/>
		{/key}
	</div>
</div>
