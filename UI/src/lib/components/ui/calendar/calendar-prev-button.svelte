<script lang="ts">
	import { Calendar as CalendarPrimitive } from "bits-ui";
	import ChevronLeftIcon from "@lucide/svelte/icons/chevron-left";
	import { buttonVariants, type ButtonVariant } from "$lib/components/ui/button/index.js";
	import { m } from '$lib/paraglide/messages.js';
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		children,
		variant = "ghost",
		...restProps
	}: CalendarPrimitive.PrevButtonProps & {
		variant?: ButtonVariant;
	} = $props();
</script>

{#snippet Fallback()}
	<ChevronLeftIcon class="size-4" />
{/snippet}

<CalendarPrimitive.PrevButton
	bind:ref
	class={cn(
		buttonVariants({ variant }),
		"size-(--cell-size) bg-transparent p-0 select-none disabled:opacity-50 rtl:rotate-180",
		className
	)}
	{...restProps}
>
	{#snippet child({ props })}
		<button {...props} aria-label={m['controls.previous_month']()}>
			{#if children}{@render children?.()}{:else}{@render Fallback()}{/if}
		</button>
	{/snippet}
</CalendarPrimitive.PrevButton>
