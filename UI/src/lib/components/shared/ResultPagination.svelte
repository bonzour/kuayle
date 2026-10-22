<script lang="ts">
	import * as Pagination from '$lib/components/ui/pagination';
	import { m } from '$lib/paraglide/messages.js';
	let { page, perPage, total, label, onchange }: { page: number; perPage: number; total: number; label: string; onchange: (page: number) => void } = $props();
</script>

{#if total > perPage}
	<Pagination.Root count={total} {perPage} {page} onPageChange={onchange} aria-label={label} class="flex flex-wrap items-center justify-end gap-2">
		<span class="text-xs text-muted-foreground"><span>{m['timesheet.page']({ page, pages: Math.ceil(total / perPage) })}</span> · {m['controls.result_count']({ count: total })}</span>
		<Pagination.PrevButton aria-label={m['timesheet.previous']()} class="min-h-8">{m['timesheet.previous']()}</Pagination.PrevButton>
		<Pagination.NextButton aria-label={m['timesheet.next']()} class="min-h-8">{m['timesheet.next']()}</Pagination.NextButton>
	</Pagination.Root>
{/if}
