<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	let { value = $bindable(''), label, disabled = false, error = '', ref = $bindable(null), onkeydown }: {
		value?: string; label: string; disabled?: boolean; error?: string;
		ref?: HTMLInputElement | null; onkeydown?: (event: KeyboardEvent) => void;
	} = $props();
	const id = $props.id();
</script>

<div class="space-y-1">
	<label for={id} class="sr-only">{label}</label>
	<input {id} type="text" bind:this={ref} {onkeydown} bind:value {disabled} aria-invalid={!!error}
		aria-describedby={`${id}-help${error ? ` ${id}-error` : ''}`}
		class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-2 py-1.5 text-sm outline-none focus:border-[var(--app-accent)] disabled:opacity-60" />
	<p id={`${id}-help`} class="text-xs text-[var(--color-text-tertiary)]">{m['efforts.planning.duration_help']()}</p>
	{#if error}<p id={`${id}-error`} role="alert" class="text-xs text-destructive">{error}</p>{/if}
</div>
