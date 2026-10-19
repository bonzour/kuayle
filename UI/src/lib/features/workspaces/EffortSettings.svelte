<script lang="ts">
	import { untrack } from 'svelte';
	import { Button } from '$lib/components/ui/button';
	import { Switch } from '$lib/components/ui/switch';
	import { getCurrentWorkspace } from './current-workspace.svelte';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import { setEffortSettings } from '$lib/api/efforts';
	import { emitAppRefresh } from '$lib/api/refresh';
	import { appToast } from '$lib/features/toast/toast';
	import type { EffortSettings, SetEffortSettingsRequest } from '$lib/types/efforts';
	import { m } from '$lib/paraglide/messages.js';

	let { slug }: { slug: string } = $props();
	const workspace = getCurrentWorkspace();
	const membership = $derived(workspace.membership);
	const settings = $derived(workspace.efforts?.slug === slug && workspace.efforts.userId === authState.user?.id
		? workspace.efforts.settings : null);
	const canManage = $derived(membership?.workspace.slug === slug && membership.userId === authState.user?.id &&
		['owner', 'admin'].includes(membership.workspace.current_user_role));
	let baseline = $state<EffortSettings | null>(null);
	let enabled = $state(true);
	let timezone = $state('Europe/Rome');
	let busy = $state(false);
	let attempt = $state<SetEffortSettingsRequest | null>(null);
	let conflict = $state(false);
	let error = $state('');
	let timezoneError = $state('');
	const dirty = $derived(!!baseline && (enabled !== baseline.enabled || timezone !== baseline.timezone));
	const stale = $derived(!!settings && !!baseline && settings.revision !== baseline.revision);

	function apply(value: EffortSettings) {
		baseline = value;
		enabled = value.enabled;
		timezone = value.timezone;
	}

	// A remote refresh may update saved values, but never overwrite a draft.
	$effect(() => {
		const value = settings;
		untrack(() => {
			if (value && (!baseline || (!dirty && !busy && !attempt && !conflict))) apply(value);
		});
	});

	async function realign() {
		busy = true;
		const value = await workspace.loadEfforts(slug);
		if (value) {
			apply(value);
			conflict = false;
			attempt = null;
			error = '';
			timezoneError = '';
		}
		busy = false;
	}

	async function save() {
		if (!canManage || !baseline || busy || conflict || stale && !attempt) return;
		attempt ??= { enabled, timezone, request_id: crypto.randomUUID(), expected_revision: baseline.revision };
		const request = attempt;
		busy = true;
		error = '';
		timezoneError = '';
		try {
			const receipt = await setEffortSettings(slug, request);
			attempt = null;
			apply({ ...baseline, enabled: request.enabled, timezone: request.timezone, revision: receipt.revision });
			const value = await workspace.loadEfforts(slug);
			if (value) apply(value);
			appToast.success(m['settings.efforts.saved']());
			emitAppRefresh(['workspace', 'issues'], slug);
		} catch (failure) {
			const result = failure as { status?: number; error?: { code?: string; details?: { field: string }[] } };
			if (result.status && result.status < 500) attempt = null;
			if (result.error?.code === 'EFFORT_REVISION_CONFLICT' || result.error?.code === 'EFFORT_REQUEST_CONFLICT') {
				conflict = true;
				error = m['settings.efforts.conflict']();
			} else if (result.error?.code === 'VALIDATION_ERROR' && result.error.details?.some(detail => detail.field === 'timezone')) {
				timezoneError = m['settings.efforts.invalid_timezone']();
			} else if (result.status === 403) {
				error = m['settings.efforts.readonly']();
				emitAppRefresh(['workspace'], slug);
			} else {
				error = attempt ? m['settings.efforts.retry_error']() : m['settings.efforts.save_error']();
			}
		} finally {
			busy = false;
		}
	}
</script>

<section class="mt-8 space-y-4 rounded-lg border border-[var(--app-border)] bg-[var(--color-bg-secondary)] p-5" aria-labelledby="efforts-title">
	<h2 id="efforts-title" class="text-base font-medium text-[var(--color-text-primary)]">Efforts</h2>
	<p id="efforts-description" class="text-xs text-[var(--color-text-tertiary)]">{m['settings.efforts.description']()}</p>
	{#if workspace.effortsFailed}
		<p role="alert" class="text-sm text-destructive">{m['settings.efforts.load_error']()}</p>
		<Button variant="outline" disabled={busy} onclick={realign}>{m['settings.efforts.reload']()}</Button>
	{:else if baseline}
		<form class="space-y-4" onsubmit={(event) => { event.preventDefault(); void save(); }}>
			<div class="flex items-center justify-between gap-4">
				<label for="efforts-enabled" class="text-sm">{m['settings.efforts.enabled']()}</label>
				<Switch id="efforts-enabled" bind:checked={enabled} disabled={!canManage || busy || !!attempt} aria-describedby="efforts-description" />
			</div>
			<div class="space-y-1">
				<label for="efforts-timezone" class="block text-sm">{m['settings.efforts.timezone']()}</label>
				<input id="efforts-timezone" type="text" bind:value={timezone} placeholder="Europe/Rome" required
					disabled={!canManage || busy || !!attempt} aria-invalid={!!timezoneError}
					aria-describedby={timezoneError ? 'efforts-timezone-error' : 'efforts-timezone-help'}
					class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-3 py-1.5 text-sm outline-none focus:border-[var(--app-accent)] disabled:opacity-60" />
				<p id="efforts-timezone-help" class="text-xs text-[var(--color-text-tertiary)]">{m['settings.efforts.timezone_help']()}</p>
				{#if timezoneError}<p id="efforts-timezone-error" role="alert" class="text-sm text-destructive">{timezoneError}</p>{/if}
			</div>
			{#if !canManage}<p class="text-xs text-[var(--color-text-tertiary)]">{m['settings.efforts.readonly']()}</p>{/if}
			{#if error}<p role="alert" class="text-sm text-destructive">{error}</p>{/if}
			{#if stale && !error}<p role="status" class="text-sm">{m['settings.efforts.conflict']()}</p>{/if}
			{#if canManage}
				<div class="flex flex-wrap gap-2">
					<Button type="submit" disabled={busy || !settings || conflict || (stale && !attempt) || (!dirty && !attempt)}>
						{busy ? m['common.saving']() : attempt ? m['settings.efforts.retry']() : m['common.save']()}
					</Button>
					{#if (conflict || stale) && !attempt}
						<Button type="button" variant="outline" disabled={busy} onclick={realign}>{m['settings.efforts.realign']()}</Button>
					{/if}
				</div>
			{/if}
		</form>
	{:else}
		<p role="status" class="text-sm text-[var(--color-text-tertiary)]">{m['settings.efforts.loading']()}</p>
	{/if}
</section>
