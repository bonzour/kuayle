<script lang="ts">
	import * as Select from '$lib/components/ui/select';
	let { id, label, value, options, onchange, placeholder = '', disabled = false, error = '' }: {
		id: string; label: string; value: string; options: { id: string; name: string }[];
		onchange: (value: string) => void; placeholder?: string; disabled?: boolean; error?: string;
	} = $props();
</script>

<div class="min-w-0 space-y-1">
	<label for={id} class="text-xs font-medium">{label}</label>
	<Select.Root type="single" {value} {disabled} onValueChange={onchange}>
		<Select.Trigger {id} aria-label={label} aria-invalid={!!error} aria-describedby={error ? `${id}-error` : undefined} class="min-h-9 w-full bg-[var(--color-bg)]">
			<span class="truncate">{options.find(option => option.id === value)?.name ?? placeholder}</span>
		</Select.Trigger>
		<Select.Content class="max-w-[calc(100vw-2rem)]">
			{#each options as option (option.id)}<Select.Item value={option.id} label={option.name}>{option.name}</Select.Item>{/each}
		</Select.Content>
	</Select.Root>
	{#if error}<p id={`${id}-error`} role="alert" class="text-xs text-destructive">{error}</p>{/if}
</div>
