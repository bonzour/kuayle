<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import LoadingState from '$lib/components/shared/LoadingState.svelte';
	import ErrorState from '$lib/components/shared/ErrorState.svelte';
	import { m } from '$lib/paraglide/messages.js';
	let { title, description, query, onquery, items, onselect, loading = false, error = '', disabled = false, searchable = true, onretry, onclose, returnFocus, context, footer, metadata, resultsLabel }: {
		title: string; description: string; query: string; onquery: (value: string) => void;
		items: { id: string; identifier: string; title: string; subtitle?: string }[];
		onselect: (id: string) => void; loading?: boolean; error?: string; disabled?: boolean; searchable?: boolean;
		onretry: () => void; onclose: () => void; returnFocus?: () => void;
		context?: Snippet; footer?: Snippet; metadata?: Snippet<[string]>; resultsLabel?: string;
	} = $props();
	const id = $props.id();
	let search = $state<HTMLInputElement | null>(null);
	let list = $state<HTMLUListElement | null>(null);
	function navigate(event: KeyboardEvent, index = -1) {
		const buttons = list?.querySelectorAll<HTMLButtonElement>('button');
		if (!buttons?.length) return;
		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();
			const next = index + (event.key === 'ArrowDown' ? 1 : -1);
			if (next < 0) search?.focus(); else buttons[Math.min(next, buttons.length - 1)]?.focus();
		} else if (index === -1 && event.key === 'Enter') { event.preventDefault(); onselect(items[0].id); }
	}
</script>

<Dialog.Root open={true} onOpenChange={(open) => { if (!open && !disabled) onclose(); }}>
	<Dialog.Content showCloseButton={!disabled} class="max-h-[85dvh] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-xl"
		onEscapeKeydown={(event) => { if (disabled) event.preventDefault(); }}
		onInteractOutside={(event) => { if (disabled) event.preventDefault(); }}
		onCloseAutoFocus={(event) => { if (returnFocus) { event.preventDefault(); returnFocus(); } }}>
		<Dialog.Header><Dialog.Title>{title}</Dialog.Title><Dialog.Description>{description}</Dialog.Description></Dialog.Header>
		<div class="min-h-0 space-y-3 overflow-y-auto">
			{@render context?.()}
			{#if searchable}
				<label for={`${id}-search`} class="block text-xs font-medium">{m['controls.issue_search']()}</label>
				<input bind:this={search} id={`${id}-search`} type="search" value={query} {disabled} aria-controls={`${id}-results`} oninput={(event) => onquery(event.currentTarget.value)} onkeydown={(event) => navigate(event)} class="min-h-10 w-full rounded-lg border border-input bg-[var(--color-bg)] px-3 text-sm focus-visible:outline-2 focus-visible:outline-ring" />
			{/if}
			<div aria-busy={loading}>
				{#if error}<ErrorState message={error} {onretry} variant="compact" />
				{:else if loading}<LoadingState variant="compact" />
				{:else if searchable}
					{#if resultsLabel}<p class="mb-2 text-xs text-muted-foreground">{resultsLabel}</p>{/if}
					<ul bind:this={list} id={`${id}-results`} class="space-y-1" aria-label={m['controls.issue_search']()}>
						{#each items as item, index (item.id)}
							<li><button type="button" {disabled} onkeydown={(event) => navigate(event, index)} onclick={() => onselect(item.id)} class="min-h-11 w-full rounded-lg border border-[var(--app-border)] p-2 text-left hover:bg-[var(--color-bg-hover)] focus-visible:bg-accent focus-visible:outline-2 focus-visible:outline-ring">
								<span class="block break-words text-sm"><span class="font-medium">{item.identifier}</span> · {item.title}</span>
								{#if item.subtitle}<span class="text-xs text-muted-foreground">{item.subtitle}</span>{/if}
								{@render metadata?.(item.id)}
							</button></li>
						{/each}
						{#if !items.length}<li class="py-4 text-sm text-muted-foreground">{m['controls.issue_empty']()}</li>{/if}
					</ul>
				{/if}
			</div>
		</div>
		<Dialog.Footer class="flex-wrap gap-2">
			{@render footer?.()}
			<Button variant="outline" {disabled} onclick={onclose}>{m['common.cancel']()}</Button>
		</Dialog.Footer>
	</Dialog.Content>
</Dialog.Root>
