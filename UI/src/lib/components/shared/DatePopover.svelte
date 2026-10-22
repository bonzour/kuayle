<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Popover from '$lib/components/ui/popover';
	import { Button } from '$lib/components/ui/button';
	import { CalendarDays, X } from 'lucide-svelte';
	import { m } from '$lib/paraglide/messages.js';

	// All date adapters share this trigger/focus/clear contract. Clear is a sibling button.
	let { open = $bindable(false), label, text, disabled = false, iconOnly = false, colorClass = '', onclear, children }: {
		open?: boolean; label: string; text: string; disabled?: boolean; iconOnly?: boolean; colorClass?: string;
		onclear?: () => void; children: Snippet;
	} = $props();
	const descriptionId = $props.id();
	let trigger = $state<HTMLButtonElement | null>(null);
	let content = $state<HTMLDivElement | null>(null);
	let pendingFocus = $state(false);
	$effect(() => {
		if (disabled) open = false;
		if (pendingFocus && !open && !disabled) { pendingFocus = false; trigger?.focus(); }
	});
</script>

<span id={descriptionId} class="sr-only">{text}</span>
<div class="inline-flex min-w-0 max-w-full items-center gap-1">
	<Popover.Root bind:open>
		<Popover.Trigger>
			{#snippet child({ props })}
				<Button {...props} bind:ref={trigger} variant="outline" size={iconOnly ? 'icon-sm' : 'sm'} {disabled} aria-label={label} aria-describedby={descriptionId} class="min-h-8 min-w-8 max-w-full font-normal">
					<CalendarDays size={16} />{#if !iconOnly}<span class="truncate {colorClass}">{text}</span>{/if}
				</Button>
			{/snippet}
		</Popover.Trigger>
		<Popover.Content bind:ref={content} align="start" class="w-auto max-w-[calc(100vw-1rem)] max-h-[min(80dvh,var(--bits-popover-content-available-height))] overflow-auto p-0"
			onOpenAutoFocus={(event) => { const day = content?.querySelector<HTMLElement>('[data-bits-day][data-focused]'); if (day) { event.preventDefault(); day.focus(); } }}
			onCloseAutoFocus={(event) => { event.preventDefault(); pendingFocus = true; }}>
			{@render children()}
		</Popover.Content>
	</Popover.Root>
	{#if onclear}
		<Button type="button" variant="ghost" size="icon-sm" class="min-h-8 min-w-8 shrink-0" {disabled} aria-label={m['controls.clear_date']()} onclick={() => { onclear?.(); open = false; trigger?.focus(); }}><X size={14} /></Button>
	{/if}
</div>
