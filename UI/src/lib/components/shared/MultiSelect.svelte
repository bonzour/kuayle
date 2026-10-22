<script lang="ts">
	import { Combobox } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import * as Popover from '$lib/components/ui/popover';
	import { Button } from '$lib/components/ui/button';
	import { Check, ChevronDown } from 'lucide-svelte';
	import { m } from '$lib/paraglide/messages.js';

	// Presentation only: callers map domain-specific All/empty values and retain authorization.
	let { label, value, options, onchange, disabled = false, emptyLabel, open = $bindable(false), onOpenChange, onremove, icon }: {
		label: string; value: string[]; options: { id: string; name: string; pinned?: boolean }[];
		onchange: (value: string[]) => void; disabled?: boolean; emptyLabel?: string; open?: boolean;
		onOpenChange?: (open: boolean) => void; onremove?: () => void; icon?: Snippet<[string]>;
	} = $props();
	let query = $state('');
	let input = $state<HTMLInputElement | null>(null);
	const selected = $derived(options.filter(option => value.includes(option.id)));
	const summary = $derived(selected.length ? `${selected[0].name}${selected.length > 1 ? ` +${selected.length - 1}` : ''}` : emptyLabel ?? m['timesheet.none_selected']());
	const filtered = $derived(options.filter(option => option.pinned || option.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())));
	$effect(() => { if (!open) query = ''; });
</script>

<Popover.Root bind:open onOpenChange={(next) => onOpenChange?.(next)}>
	<Popover.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="outline" size="sm" {disabled} aria-label={label} data-slot="select-trigger" class="min-h-8 w-full min-w-0 justify-between bg-[var(--color-bg)] font-normal">
				<span class="truncate">{summary}</span><ChevronDown size={14} class="shrink-0" />
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content data-slot="select-content" align="start" class="w-72 max-w-[calc(100vw-2rem)] p-2" onOpenAutoFocus={(event) => { event.preventDefault(); input?.focus(); }}>
		{#if open}
		<Combobox.Root type="multiple" {value} onValueChange={onchange} open={true} onOpenChange={(next) => { if (!next) open = false; }} {disabled}>
			<Combobox.Input bind:ref={input} aria-label={`${m['controls.search']()} · ${label}`} placeholder={m['controls.search']()} oninput={(event) => query = event.currentTarget.value} class="mb-2 min-h-9 w-full rounded-md border border-input bg-background px-2 text-sm focus-visible:outline-2 focus-visible:outline-ring" />
			<p class="mb-2 max-h-20 overflow-auto break-words text-xs text-muted-foreground">{selected.length ? selected.map(option => option.name).join(', ') : summary}</p>
			<Combobox.ContentStatic class="max-h-64 overflow-y-auto" aria-label={label}>
				{#each filtered as option (option.id)}
					<Combobox.Item value={option.id} label={option.name} class="flex min-h-9 cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none data-highlighted:bg-accent data-highlighted:text-accent-foreground">
						<span class="w-4 shrink-0" aria-hidden="true">{#if value.includes(option.id)}<Check size={14} />{/if}</span>
						{@render icon?.(option.id)}<span class="break-words">{option.name}</span>
					</Combobox.Item>
				{/each}
				{#if !filtered.some(option => !option.pinned)}<p class="p-2 text-sm text-muted-foreground">{m['controls.no_results']()}</p>{/if}
			</Combobox.ContentStatic>
		</Combobox.Root>
		{/if}
		{#if onremove}<Button variant="ghost" size="sm" class="mt-2 min-h-8" onclick={() => { open = false; onremove?.(); }}>{m['sharedComponents.filter_builder.remove_filter']()}</Button>{/if}
	</Popover.Content>
</Popover.Root>
