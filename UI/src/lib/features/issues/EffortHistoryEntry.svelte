<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { EffortEvent, EffortSnapshot } from '$lib/types/efforts';
	import { formatRelativeTime } from '$lib/utils/format';
	import { getLocale } from '$lib/paraglide/runtime.js';
	import { m } from '$lib/paraglide/messages.js';
	import { formatEffortMinutes } from './effort-duration';
	let { event, actions }: { event: EffortEvent; actions?: Snippet } = $props();
	const name = (id: string) => {
		const user = event.users?.find((user) => user.id === id);
		return user?.available
			? user.display_name || user.name || m['efforts.worklogs.unavailable_user']()
			: m['efforts.worklogs.unavailable_user']();
	};
	const actor = $derived(
		event.actor.kind === 'system' ? m['efforts.history.system']({ source: event.actor.system_source }) : name(event.actor.user_id)
	);
	const label = $derived(
		{
			estimate_set: m['efforts.history.estimate'](),
			remaining_set: m['efforts.history.remaining'](),
			worklog_created: m['efforts.history.created'](),
			worklog_updated: m['efforts.history.updated'](),
			worklog_deleted: m['efforts.history.deleted']()
		}[event.event_type]
	);
	const cause = $derived(
		{
			manual: '',
			worklog: m['efforts.history.cause_worklog'](),
			closure: m['efforts.history.cause_closure'](),
			reactivation: m['efforts.history.cause_reactivation']()
		}[event.cause]
	);
	function minutes(snapshot: EffortSnapshot | null) {
		if (!snapshot || !('planning' in snapshot)) return m['efforts.history.absent']();
		const value = event.event_type === 'estimate_set' ? snapshot.planning.estimate_minutes : snapshot.planning.remaining_minutes;
		return value === null ? m['efforts.history.absent']() : formatEffortMinutes(value);
	}
</script>

<div class="min-w-0 flex-1 space-y-1 text-xs text-[var(--color-text-tertiary)]" data-effort-event={event.event_type}>
	<div class="flex items-start justify-between gap-2">
		<p>
			<strong class="text-[var(--color-text-secondary)]">{actor}</strong>
			{label}{#if cause}
				· {cause}{/if} · <time datetime={event.created_at}>{formatRelativeTime(event.created_at, getLocale())}</time>
		</p>
		{#if actions}<div class="flex shrink-0 items-center gap-1">{@render actions()}</div>{/if}
	</div>
	{#if event.event_type === 'estimate_set' || event.event_type === 'remaining_set'}
		<p>
			<span aria-label={m['efforts.history.before']()}>{minutes(event.before)}</span> →
			<span aria-label={m['efforts.history.after']()}>{minutes(event.after)}</span>
		</p>
	{:else}
		<details>
			<summary class="cursor-pointer hover:text-[var(--color-text-secondary)]">{m['efforts.history.details']()}</summary>
			<div class="mt-1 space-y-2 border-l border-[var(--app-border)] pl-2">
				{#each [{ label: m['efforts.history.before'](), snapshot: event.before }, { label: m['efforts.history.after'](), snapshot: event.after }] as side}
					{#if side.snapshot && 'worklog' in side.snapshot}
						{@const worklog = side.snapshot.worklog}
						<div>
							<p class="font-medium">{side.label}</p>
							<p>
								{name(worklog.owner_user_id)} · <time datetime={worklog.work_date}>{worklog.work_date}</time> · {formatEffortMinutes(
									worklog.duration_minutes
								)}
							</p>
							{#if worklog.created_by_user_id !== worklog.owner_user_id}<p>
									{m['efforts.worklogs.inserted_by']({ name: name(worklog.created_by_user_id) })}
								</p>{/if}
							{#if worklog.description}<p class="whitespace-pre-wrap break-words">{worklog.description}</p>{/if}
						</div>
					{/if}
				{/each}
			</div>
		</details>
	{/if}
</div>
