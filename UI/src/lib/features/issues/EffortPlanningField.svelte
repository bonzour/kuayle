<script lang="ts">
	import { tick, untrack } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { setEffortEstimate, setEffortRemaining } from '$lib/api/efforts';
	import { emitAppRefresh } from '$lib/api/refresh';
	import type { EffortSummary } from '$lib/types/efforts';
	import { m } from '$lib/paraglide/messages.js';
	import EffortDurationInput from './EffortDurationInput.svelte';
	import { parseEffortMinutes, formatEffortMinutes } from './effort-duration';

	let { kind, slug, identifier, summary, canEdit, reload }: {
		kind: 'estimate' | 'remaining'; slug: string; identifier: string; summary: EffortSummary;
		canEdit: boolean; reload: () => Promise<EffortSummary | null>;
	} = $props();
	let baseline = $state<{ value: number | null; revision: number } | null>(null);
	let text = $state('');
	let busy = $state(false);
	let editing = $state(false);
	let inputRef = $state<HTMLInputElement | null>(null);
	let valueRef = $state<HTMLButtonElement | null>(null);
	let attempt = $state<{ request_id: string; expected_revision: number; minutes: number | null } | null>(null);
	let conflict = $state(false);
	let denied = $state(false);
	let error = $state('');
	let inputError = $state('');
	const label = $derived(kind === 'estimate' ? m['efforts.planning.estimate']() : m['efforts.planning.remaining']());
	const baselineText = $derived(baseline?.value == null ? (kind === 'remaining' ? '0m' : '') : formatEffortMinutes(baseline.value));
	const dirty = $derived(text !== baselineText);
	const stale = $derived(!!baseline && baseline.revision !== summary.revision);
	const displayValue = $derived(baseline?.value == null
		? (kind === 'estimate' ? m['efforts.planning.unestimated']() : `0m (${m['efforts.planning.unset']()})`)
		: formatEffortMinutes(baseline.value));

	async function edit() {
		editing = true;
		await tick();
		inputRef?.focus();
		inputRef?.select();
	}

	async function cancel() {
		if (!attempt && !conflict && !error) {
			text = baselineText;
			inputError = '';
		}
		editing = false;
		await tick();
		valueRef?.focus();
	}

	function focusout(event: FocusEvent) {
		if (editing && !(event.currentTarget as HTMLElement).contains(event.relatedTarget as Node | null) && !attempt) void save();
	}

	function keydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			void save();
		} else if (event.key === 'Escape') {
			event.preventDefault();
			event.stopPropagation();
			void cancel();
		}
	}

	function apply(value: EffortSummary) {
		baseline = { value: kind === 'estimate' ? value.own.estimate_minutes : value.own.remaining_minutes, revision: value.revision };
		text = baseline.value === null ? (kind === 'remaining' ? '0m' : '') : formatEffortMinutes(baseline.value);
	}

	$effect(() => {
		const value = summary;
		untrack(() => {
			if (!baseline || (!dirty && !busy && !attempt && !conflict && !error)) apply(value);
		});
	});

	async function realign() {
		busy = true;
		const value = await reload();
		if (value) {
			apply(value);
			attempt = null;
			conflict = false;
			denied = false;
			error = '';
			inputError = '';
			editing = false;
		}
		busy = false;
	}

	async function save(remove = false) {
		if (!canEdit || !baseline || busy || conflict || denied || (stale && !attempt)) return;
		if (!attempt) {
			const minutes = remove ? null : parseEffortMinutes(text);
			if (minutes === null && !remove) {
				inputError = m['efforts.planning.invalid_duration']();
				return;
			}
			if (minutes === baseline.value) { inputError = ''; editing = false; return; }
			attempt = { request_id: crypto.randomUUID(), expected_revision: baseline.revision, minutes };
		}
		const request = attempt;
		busy = true;
		error = '';
		inputError = '';
		try {
			const mutation = { request_id: request.request_id, expected_revision: request.expected_revision };
			const receipt = kind === 'estimate'
				? await setEffortEstimate(slug, identifier, { ...mutation, estimate_minutes: request.minutes })
				: await setEffortRemaining(slug, identifier, { ...mutation, remaining_minutes: request.minutes! });
			attempt = null;
			baseline = { value: request.minutes, revision: receipt.revision };
			text = request.minutes === null ? '' : formatEffortMinutes(request.minutes);
			const value = await reload();
			if (value) apply(value);
			editing = false;
			emitAppRefresh(['issues'], slug);
		} catch (failure) {
			const result = failure as { status?: number; error?: { code?: string } };
			if (result.status && result.status < 500) attempt = null;
			if (result.error?.code === 'EFFORT_REVISION_CONFLICT' || result.error?.code === 'EFFORT_REQUEST_CONFLICT') {
				conflict = true;
				error = m['efforts.planning.conflict']();
			} else if (result.error?.code === 'EFFORT_DISABLED') {
				error = m['efforts.planning.disabled']();
				emitAppRefresh(['workspace'], slug);
				await reload();
			} else if (result.status === 403 || result.status === 404) {
				denied = true;
				error = result.status === 403 ? m['efforts.planning.readonly']() : m['efforts.planning.unavailable']();
				emitAppRefresh(['workspace'], slug);
				await reload();
			} else {
				error = attempt ? m['efforts.planning.retry_error']() : m['efforts.planning.save_error']();
				if (result.status === 400) await reload();
			}
		} finally {
			busy = false;
		}
	}
</script>

{#snippet feedback()}
	{#if error}<p role="alert" class="px-2 py-1 text-xs text-destructive">{error}</p>{/if}
	{#if stale && !error}<p role="status" class="px-2 py-1 text-xs">{m['efforts.planning.conflict']()}</p>{/if}
	{#if attempt}
		<Button type="button" size="sm" variant="outline" disabled={!canEdit || busy || denied || conflict} onclick={() => save()}>{m['settings.efforts.retry']()}</Button>
	{:else if conflict || stale || error}
		<Button type="button" size="sm" variant="outline" disabled={busy} onclick={realign}>{m['settings.efforts.realign']()}</Button>
	{/if}
{/snippet}

<div role="group" aria-label={label}>
	<div class="flex items-start gap-3 rounded-md px-2 py-1.5 hover:bg-[var(--color-bg-hover)] transition-colors">
		<span class="w-20 shrink-0 text-xs leading-5 text-[var(--color-text-tertiary)]">{label}</span>
		<div class="min-w-0 flex-1">
			{#if editing}
				<div onfocusout={focusout}>
					<EffortDurationInput bind:value={text} bind:ref={inputRef} {label}
						disabled={!canEdit || busy || !!attempt || denied} error={inputError} onkeydown={keydown} />
					{#if kind === 'estimate' && baseline?.value !== null}
						<button type="button" class="mt-1 rounded-sm py-1 text-left text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] disabled:opacity-50"
							disabled={!canEdit || busy || denied || conflict || stale || !!attempt} onclick={() => save(true)}>{m['efforts.planning.remove_estimate']()}</button>
					{/if}
				</div>
			{:else}
				<button bind:this={valueRef} type="button" aria-label={label} disabled={!canEdit || busy || denied} onclick={edit}
					class="flex items-center gap-1.5 text-left text-sm text-[var(--color-text-primary)] disabled:cursor-default">
					{displayValue}
				</button>
			{/if}
		</div>
	</div>
	{@render feedback()}
</div>
