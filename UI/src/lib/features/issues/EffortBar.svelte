<script lang="ts">
	let { kind, minutes, scale }: {
		kind: 'estimate' | 'spent' | 'remaining'; minutes: number | null; scale: number;
	} = $props();
	const color = $derived({ estimate: 'var(--app-accent)', spent: 'var(--color-success)', remaining: 'var(--color-warning)' }[kind]);
</script>

<div aria-hidden="true" data-effort-bar={kind} class="h-1.5 overflow-hidden rounded-full bg-[var(--app-border)]">
	<div class="h-full rounded-full" style:width={`${(minutes ?? 0) / scale * 100}%`} style:background-color={color}></div>
</div>
