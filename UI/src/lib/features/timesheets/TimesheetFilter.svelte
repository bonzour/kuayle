<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	import TeamIcon from '$lib/components/shared/TeamIcon.svelte';
	import { m } from '$lib/paraglide/messages.js';
	import { fromSelectValues } from './selection';
	import type { TimesheetSelection, TimesheetTeamOption } from '$lib/types/timesheets';

	let { label, selection, options, disabled = false, onchange }: {
		label: string;
		selection: TimesheetSelection;
		options: { id: string; name: string; team?: TimesheetTeamOption }[];
		disabled?: boolean;
		onchange: (value: TimesheetSelection) => void;
	} = $props();
</script>

<div class="min-w-0 space-y-1">
	<p class="text-xs text-[var(--color-text-secondary)]">{label}</p>
	<Select.Root type="multiple" value={selection ?? ['all']} {disabled} onValueChange={(values) => onchange(fromSelectValues(values, selection))}>
		<Select.Trigger size="sm" aria-label={label} class="w-full min-w-[180px] bg-[var(--color-bg)]">
			<span class="truncate">
				{selection === null ? m['timesheet.all']() : selection.length === 0 ? m['timesheet.none_selected']() : m['timesheet.selected_count']({ count: selection.length })}
			</span>
		</Select.Trigger>
		<Select.Content>
			<Select.Item value="all" label={m['timesheet.all']()}>{m['timesheet.all']()}</Select.Item>
			{#each options as option (option.id)}
				<Select.Item value={option.id} label={option.name}>
					{#if option.team}<TeamIcon team={option.team} size={14} />{/if}
					{option.name}
				</Select.Item>
			{/each}
		</Select.Content>
	</Select.Root>
</div>
