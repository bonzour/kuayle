<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { m } from '$lib/paraglide/messages.js';
	let { message, retryLabel, onretry, variant = 'page' }: { message?: string; retryLabel?: string; onretry?: () => void; variant?: 'page' | 'compact' } = $props();
</script>
<div class="flex flex-col items-center justify-center gap-3 {variant === 'page' ? 'min-h-64' : 'min-h-24 py-4'}">
	<p role="alert" class="text-sm text-destructive">{message ?? m['controls.failed']()}</p>
	{#if onretry}<Button variant="outline" onclick={onretry}>{retryLabel ?? m['controls.retry']()}</Button>{/if}
</div>
