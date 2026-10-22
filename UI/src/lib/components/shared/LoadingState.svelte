<script lang="ts">
	import { m } from '$lib/paraglide/messages.js';
	let { message, variant = 'page' }: { message?: string; variant?: 'page' | 'compact' } = $props();
</script>
<div role="status" class="flex flex-col items-center justify-center gap-3 {variant === 'page' ? 'min-h-64' : 'min-h-24 py-4'}">
	<div aria-hidden="true" class="h-5 w-5 animate-spin rounded-full border-2 border-[var(--color-text-tertiary)] border-t-transparent"></div>
	<p class="text-sm text-[var(--color-text-secondary)]">{message ?? m['common.loading']()}</p>
</div>
