<script lang="ts">
	import MultiSelect from '$lib/components/shared/MultiSelect.svelte';
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
	<MultiSelect {label} value={selection ?? ['all']} options={[{ id: 'all', name: m['timesheet.all'](), pinned: true }, ...options]} {disabled} onchange={(values) => onchange(fromSelectValues(values, selection))}>
		{#snippet icon(id)}{@const team = options.find(option => option.id === id)?.team}{#if team}<TeamIcon {team} size={14} />{/if}{/snippet}
	</MultiSelect>
</div>
