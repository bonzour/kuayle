<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
	import { createEffortWorklog, updateEffortWorklog, deleteEffortWorklog, getEffortSettings, listEffortWorklogOwners, listEffortWorklogs } from '$lib/api/efforts';
	import { emitAppRefresh } from '$lib/api/refresh';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import type { CreateEffortWorklogRequest, UpdateEffortWorklogRequest, DeleteEffortWorklogRequest, EffortUser, EffortWorklog } from '$lib/types/efforts';
	import { m } from '$lib/paraglide/messages.js';
	import EffortDurationInput from './EffortDurationInput.svelte';
	import { formatEffortMinutes } from './effort-duration';
	import { validateWorklogDraft, worklogPeriodDays } from './effort-worklog';

	let { slug, identifier, mode, worklog, revision, userId, userName, admin, canWrite, onclose, onsaved }: {
		slug: string; identifier: string; mode: 'create' | 'update' | 'delete'; worklog: EffortWorklog | null;
		revision: number; userId: string; userName: string; admin: boolean; canWrite: boolean;
		onclose: () => void; onsaved: () => void;
	} = $props();
	type Attempt = { mode: 'create'; request: CreateEffortWorklogRequest } | { mode: 'update'; request: UpdateEffortWorklogRequest } | { mode: 'delete'; request: DeleteEffortWorklogRequest };
	let baselineRevision = $state(untrack(() => revision));
	let currentWorklog = $state(untrack(() => worklog));
	let owner = $state(untrack(() => worklog?.owner.id ?? userId));
	let date = $state(untrack(() => worklog?.work_date ?? ''));
	let endDate = $state('');
	let skipWeekends = $state(true);
	let duration = $state(untrack(() => worklog ? formatEffortMinutes(worklog.duration_minutes) : ''));
	let description = $state(untrack(() => worklog?.description ?? ''));
	let today = $state('');
	const periodDays = $derived(mode === 'create' && date <= today && endDate <= today ? worklogPeriodDays(date, endDate, skipWeekends) : 0);
	const dailyMinutes = $derived(validateWorklogDraft(date, duration, description, today).minutes);
	let enabled = $state(false);
	let owners = $state<EffortUser[]>([]);
	let ready = $state(false);
	let busy = $state(false);
	let denied = $state(false);
	let conflict = $state(false);
	let attempt = $state<Attempt | null>(null);
	let error = $state('');
	let errors = $state<Record<string, string>>({});
	let disposed = false;
	const current = () => !disposed && authState.user?.id === userId;
	const locked = $derived(busy || !!attempt || !ready || !enabled || !canWrite || denied);
	const id = $props.id();
	const title = $derived(mode === 'create' ? m['efforts.worklogs.create']() : mode === 'update' ? m['efforts.worklogs.edit']() : m['efforts.worklogs.delete_title']());
	const name = (user: EffortUser) => user.available ? user.display_name || user.name || m['efforts.worklogs.unavailable_user']() : m['efforts.worklogs.unavailable_user']();

	async function initialize() {
		try {
			const [settings, candidates] = await Promise.all([getEffortSettings(slug), mode === 'create' && admin ? listEffortWorklogOwners(slug, identifier) : Promise.resolve([])]);
			if (!current()) return false;
			today = settings.today;
			enabled = settings.enabled;
			owners = candidates;
			if (!date) date = today;
			if (!endDate) endDate = date;
			ready = true;
			if (!enabled) error = m['efforts.planning.disabled']();
			return true;
		} catch (failure) {
			if (!current()) return false;
			const status = (failure as { status?: number }).status;
			denied = status === 403 || status === 404;
			error = status === 403 ? m['efforts.worklogs.denied']() : status === 404 ? m['efforts.planning.unavailable']() : m['efforts.worklogs.load_error']();
			return false;
		}
	}

	onMount(() => {
		void initialize();
		return () => { disposed = true; };
	});

	async function realign() {
		busy = true;
		error = '';
		try {
			if (!await initialize() || !current()) return;
			let page = await listEffortWorklogs(slug, identifier);
			if (!current()) return;
			let fresh = page.data.find(row => row.id === currentWorklog?.id);
			while (mode !== 'create' && !fresh && page.has_more) {
				page = await listEffortWorklogs(slug, identifier, page.page + 1);
				if (!current()) return;
				fresh = page.data.find(row => row.id === currentWorklog?.id);
			}
			if (mode !== 'create' && !fresh) {
				denied = true;
				error = m['efforts.worklogs.missing']();
				return;
			}
			currentWorklog = fresh ?? null;
			baselineRevision = page.revision;
			owner = fresh?.owner.id ?? userId;
			date = fresh?.work_date ?? today;
			endDate = date;
			skipWeekends = true;
			duration = fresh ? formatEffortMinutes(fresh.duration_minutes) : '';
			description = fresh?.description ?? '';
			conflict = false;
			denied = false;
			errors = {};
		} catch {
			if (current()) error = m['efforts.worklogs.load_error']();
		} finally {
			if (current()) busy = false;
		}
	}

	async function save() {
		if (busy || !ready || conflict || (!attempt && (!canWrite || !enabled || denied))) return;
		if (!attempt) {
			errors = {};
			const mutation = { request_id: crypto.randomUUID(), expected_revision: baselineRevision };
			if (mode === 'delete') attempt = { mode, request: mutation };
			else {
				const valid = validateWorklogDraft(date, duration, description, today);
				if (mode === 'create' && (!validateWorklogDraft(endDate, duration, description, today).dateValid || endDate < date)) errors.end_date = m['efforts.worklogs.invalid_period']();
				else if (mode === 'create' && !periodDays) errors.end_date = m['efforts.worklogs.empty_period']();
				if (!valid.dateValid) errors.work_date = m['efforts.worklogs.invalid_date']();
				if (!valid.durationValid) errors.duration_minutes = m['efforts.worklogs.invalid_duration']();
				if (!valid.descriptionValid) errors.description = m['efforts.worklogs.invalid_description']();
				if (mode === 'create' && admin && !owners.some(user => user.id === owner)) errors.owner_user_id = m['efforts.worklogs.invalid_owner']();
				if (Object.keys(errors).length) return;
				const request = { ...mutation, work_date: date, duration_minutes: valid.minutes!, description: description || null };
				attempt = mode === 'create' ? { mode, request: { ...request, owner_user_id: owner, end_date: endDate, skip_weekends: skipWeekends } } : { mode, request };
			}
		}
		const submission = attempt;
		busy = true;
		error = '';
		try {
			switch (submission.mode) {
				case 'create': await createEffortWorklog(slug, identifier, submission.request); break;
				case 'update': await updateEffortWorklog(slug, identifier, currentWorklog!.id, submission.request); break;
				case 'delete': await deleteEffortWorklog(slug, identifier, currentWorklog!.id, submission.request); break;
			}
			if (current()) { attempt = null; onsaved(); }
		} catch (failure) {
			if (!current()) return;
			const result = failure as { status?: number; error?: { code?: string; details?: { field?: string }[] } };
			if (result.status && result.status < 500) attempt = null;
			const code = result.error?.code;
			if (code === 'EFFORT_REVISION_CONFLICT' || code === 'EFFORT_REQUEST_CONFLICT') {
				conflict = true;
				error = m['efforts.worklogs.conflict']();
			} else if (code === 'EFFORT_DAILY_LIMIT') {
				errors.duration_minutes = m['efforts.worklogs.daily_limit']();
			} else if (code === 'VALIDATION_ERROR') {
				for (const detail of result.error?.details ?? []) {
					if (detail.field === 'end_date') errors.end_date = m['efforts.worklogs.invalid_period']();
					if (detail.field === 'work_date') errors.work_date = m['efforts.worklogs.invalid_date']();
					if (detail.field === 'duration_minutes') errors.duration_minutes = m['efforts.worklogs.invalid_duration']();
					if (detail.field === 'description') errors.description = m['efforts.worklogs.invalid_description']();
					if (detail.field === 'owner_user_id') errors.owner_user_id = m['efforts.worklogs.invalid_owner']();
				}
				if (!Object.keys(errors).length) error = m['efforts.worklogs.save_error']();
			} else if (code === 'EFFORT_DISABLED') {
				enabled = false;
				error = m['efforts.planning.disabled']();
				emitAppRefresh(['workspace'], slug);
			} else if (result.status === 403 || result.status === 404 || result.status === 401) {
				denied = true;
				error = result.status === 403 ? m['efforts.worklogs.denied']() : m['efforts.planning.unavailable']();
				emitAppRefresh(['workspace', 'issues'], slug);
			} else error = attempt ? m['efforts.planning.retry_error']() : m['efforts.worklogs.save_error']();
		} finally {
			if (current()) busy = false;
		}
	}
</script>

<Dialog.Root open={true} onOpenChange={(open) => { if (!open && !busy && !attempt) onclose(); }}>
	<Dialog.Content showCloseButton={!busy && !attempt} class="max-h-[80vh] overflow-y-auto"
		onEscapeKeydown={(event) => { if (busy || attempt) event.preventDefault(); }}
		onInteractOutside={(event) => { if (busy || attempt) event.preventDefault(); }}>
		<Dialog.Header>
			<Dialog.Title>{title}</Dialog.Title>
			<Dialog.Description>{mode === 'delete' ? m['efforts.worklogs.delete_description']() : m['efforts.worklogs.form_description']()}</Dialog.Description>
		</Dialog.Header>
		<form class="space-y-3" novalidate onsubmit={(event) => { event.preventDefault(); void save(); }}>
			{#if mode !== 'delete'}
				<div class="space-y-1">
					<label for={`${id}-owner`} class="text-xs font-medium">{m['efforts.worklogs.owner']()}</label>
					{#if mode === 'create' && admin}
						<select id={`${id}-owner`} bind:value={owner} disabled={locked} aria-invalid={!!errors.owner_user_id} aria-describedby={errors.owner_user_id ? `${id}-owner-error` : undefined} class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-2 py-2">
							{#each owners as user}<option value={user.id}>{name(user)}</option>{/each}
						</select>
					{:else}<input id={`${id}-owner`} readonly value={currentWorklog ? name(currentWorklog.owner) : userName} class="w-full bg-transparent py-1" />{/if}
					{#if errors.owner_user_id}<p id={`${id}-owner-error`} role="alert" class="text-xs text-destructive">{errors.owner_user_id}</p>{/if}
				</div>
				<div class="grid gap-3 {mode === 'create' ? 'grid-cols-2' : ''}">
				<div class="space-y-1">
					<label for={`${id}-date`} class="text-xs font-medium">{mode === 'create' ? m['efforts.worklogs.from']() : m['efforts.worklogs.date']()}</label>
					<input id={`${id}-date`} type="date" bind:value={date} max={today || undefined} min="0001-01-01" disabled={locked} aria-invalid={!!errors.work_date} aria-describedby={errors.work_date ? `${id}-date-error` : undefined} class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-2 py-2" />
					{#if errors.work_date}<p id={`${id}-date-error`} role="alert" class="text-xs text-destructive">{errors.work_date}</p>{/if}
				</div>
				{#if mode === 'create'}
					<div class="space-y-1">
						<label for={`${id}-end`} class="text-xs font-medium">{m['efforts.worklogs.to']()}</label>
						<input id={`${id}-end`} type="date" bind:value={endDate} min={date || '0001-01-01'} max={today || undefined} disabled={locked} aria-invalid={!!errors.end_date} aria-describedby={errors.end_date ? `${id}-end-error` : undefined} class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-2 py-2" />
						{#if errors.end_date}<p id={`${id}-end-error`} role="alert" class="text-xs text-destructive">{errors.end_date}</p>{/if}
					</div>
				{/if}
				</div>
				{#if mode === 'create'}
					<label class="flex items-center gap-2 text-xs">
						<input type="checkbox" bind:checked={skipWeekends} disabled={locked} />{m['efforts.worklogs.skip_weekends']()}
					</label>
				{/if}
				<div class="space-y-1">
					<p class="text-xs font-medium">{mode === 'create' ? m['efforts.worklogs.daily_duration']() : m['efforts.worklogs.duration']()}</p>
					<EffortDurationInput bind:value={duration} label={m['efforts.worklogs.duration']()} disabled={locked} error={errors.duration_minutes ?? ''} />
				</div>
				{#if mode === 'create' && periodDays && dailyMinutes !== null && dailyMinutes > 0 && dailyMinutes <= 1440}
					<p role="status" class="text-xs text-[var(--color-text-secondary)]">{m['efforts.worklogs.period_summary']({ count: periodDays, total: formatEffortMinutes(dailyMinutes * periodDays) })}</p>
				{/if}
				<div class="space-y-1">
					<label for={`${id}-description`} class="text-xs font-medium">{m['efforts.worklogs.description']()}</label>
					<textarea id={`${id}-description`} bind:value={description} rows="4" disabled={locked} aria-invalid={!!errors.description} aria-describedby={`${id}-count${errors.description ? ` ${id}-description-error` : ''}`} class="w-full rounded-md border border-[var(--app-border)] bg-[var(--color-bg)] px-2 py-2"></textarea>
					<p id={`${id}-count`} class="text-xs text-[var(--color-text-tertiary)]">{Array.from(description).length} / 2000</p>
					{#if errors.description}<p id={`${id}-description-error`} role="alert" class="text-xs text-destructive">{errors.description}</p>{/if}
				</div>
			{:else if currentWorklog}
				<p>{currentWorklog.work_date} · {name(currentWorklog.owner)} · {formatEffortMinutes(currentWorklog.duration_minutes)}</p>
				{#if currentWorklog.description}<p class="whitespace-pre-wrap break-words">{currentWorklog.description}</p>{/if}
			{/if}
			{#if !ready && !error}<p role="status">{m['settings.efforts.loading']()}</p>{/if}
			{#if error}<p role="alert" class="text-sm text-destructive">{error}</p>{/if}
			{#if conflict || denied || (error && (!ready || !enabled))}
				<Button type="button" size="sm" variant="outline" disabled={busy || !!attempt} onclick={realign}>{m['settings.efforts.realign']()}</Button>
			{/if}
			<Dialog.Footer>
				<Button type="button" variant="outline" disabled={busy || !!attempt} onclick={onclose}>{m['common.cancel']()}</Button>
				<Button type="submit" variant={mode === 'delete' ? 'destructive' : 'default'} disabled={busy || !ready || conflict || (!attempt && (!canWrite || !enabled || denied))}>
					{attempt ? m['settings.efforts.retry']() : mode === 'delete' ? m['common.delete']() : m['common.save']()}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
