<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { Button } from '$lib/components/ui/button';
	import { Ellipsis, Pencil, Trash2 } from 'lucide-svelte';
	import { m } from '$lib/paraglide/messages.js';
	let { onedit, ondelete }: { onedit: () => void; ondelete: () => void } = $props();
	let actionChosen = false;
	let trigger = $state<HTMLButtonElement | null>(null);
	// Put focus on the durable trigger before the adapter captures the dialog's return target.
	function run(action: () => void) { actionChosen = true; trigger?.focus(); action(); }
</script>

<DropdownMenu.Root onOpenChange={(open) => { if (open) actionChosen = false; }}>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}<Button {...props} bind:ref={trigger} variant="ghost" size="icon-sm" class="min-h-8 min-w-8" aria-label={m['controls.worklog_actions']()}><Ellipsis size={16} /></Button>{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" onCloseAutoFocus={(event) => { if (actionChosen) event.preventDefault(); }}>
		<DropdownMenu.Item onSelect={() => run(onedit)}><Pencil size={14} />{m['efforts.worklogs.edit']()}</DropdownMenu.Item>
		<DropdownMenu.Item variant="destructive" onSelect={() => run(ondelete)}><Trash2 size={14} />{m['common.delete']()}</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
