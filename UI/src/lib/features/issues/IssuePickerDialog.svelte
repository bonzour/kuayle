<script lang="ts">
	import { untrack } from 'svelte';
	import type { Issue } from '$lib/types/issue';
	import { listIssues } from '$lib/api/issues';
	import { authState } from '$lib/features/auth/auth.state.svelte';
	import IssueStatusIcon from './IssueStatusIcon.svelte';
	import IssuePriorityIcon from './IssuePriorityIcon.svelte';
	import IssueSearchDialog from './IssueSearchDialog.svelte';
	import { m } from '$lib/paraglide/messages.js';
	let { open = $bindable(false), slug, title, description, actionLabel, excludeIds = [], onselect, returnFocus }: {
		open: boolean; slug: string; title?: string; description?: string; actionLabel?: string;
		returnFocus?: () => void; excludeIds?: string[]; onselect: (issue: Issue) => void | Promise<void>;
	} = $props();
	interface HighlightSegment { text: string; match: boolean; }
	let searchQuery = $state('');
	let searchResults = $state<Issue[]>([]);
	let searching = $state(false);
	let selecting = $state(false);
	let error = $state('');
	let generation = 0;
	let origin: HTMLElement | null = null;
	const canSearchIssues = (value: string) => value.trim().length >= 2 || /[\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff]/.test(value);
	async function load() {
		const version = ++generation;
		const user = authState.user?.id;
		const workspace = slug;
		const current = () => open && version === generation && slug === workspace && authState.user?.id === user;
		searching = true; error = ''; searchResults = [];
		try {
			const response = await listIssues(workspace, canSearchIssues(searchQuery) ? { search: searchQuery, per_page: '12' } : { sort: 'updated_at', order: 'desc', per_page: '12' });
			if (current()) searchResults = response.data.filter(issue => !excludeIds.includes(issue.id));
		} catch { if (current()) error = m['controls.failed'](); }
		finally { if (current()) searching = false; }
	}
	$effect(() => {
		void slug; void searchQuery; void excludeIds; void authState.user?.id;
		if (!open) { untrack(() => { searchQuery = ''; searchResults = []; }); return; }
		untrack(() => { if (!origin?.isConnected) origin = document.activeElement as HTMLElement; generation++; searchResults = []; searching = true; });
		const timer = setTimeout(() => { void load(); }, canSearchIssues(searchQuery) ? 250 : 0);
		return () => { clearTimeout(timer); generation++; };
	});
	async function selectIssue(id: string) {
		const issue = searchResults.find(issue => issue.id === id);
		if (!issue || selecting) return;
		selecting = true;
		try { await onselect(issue); open = false; }
		catch { error = m['controls.failed'](); }
		finally { selecting = false; }
	}
	function plainText(html: string | null | undefined) {
		if (!html) return '';

		if (typeof document !== 'undefined') {
			const el = document.createElement('div');
			el.innerHTML = html;
			return (el.textContent ?? '').replace(/\s+/g, ' ').trim();
		}

		return html
			.replace(/<[^>]*>/g, ' ')
			.replace(/&nbsp;/g, ' ')
			.replace(/&amp;/g, '&')
			.replace(/&lt;/g, '<')
			.replace(/&gt;/g, '>')
			.replace(/&quot;/g, '"')
			.replace(/&#39;/g, "'")
			.replace(/\s+/g, ' ')
			.trim();
	}

	function highlightedSegments(value: string | null | undefined, query: string): HighlightSegment[] {
		const text = value ?? '';
		const term = query.trim();
		if (!text || !term) return [{ text, match: false }];

		const lowerText = text.toLowerCase();
		const lowerTerm = term.toLowerCase();
		const segments: HighlightSegment[] = [];
		let cursor = 0;
		let index = lowerText.indexOf(lowerTerm);

		while (index !== -1) {
			if (index > cursor) {
				segments.push({ text: text.slice(cursor, index), match: false });
			}
			segments.push({ text: text.slice(index, index + term.length), match: true });
			cursor = index + term.length;
			index = lowerText.indexOf(lowerTerm, cursor);
		}

		if (cursor < text.length) {
			segments.push({ text: text.slice(cursor), match: false });
		}

		return segments;
	}

	function highlightClass(match: boolean) {
		return match ? 'rounded-sm bg-yellow-300 px-0.5 text-slate-950' : '';
	}

	function descriptionSnippet(description: string | null, query: string) {
		const text = plainText(description);
		const term = query.trim();
		if (!text || !term) return '';

		const matchIndex = text.toLowerCase().indexOf(term.toLowerCase());
		if (matchIndex === -1) return '';

		const context = 56;
		const start = Math.max(0, matchIndex - context);
		const end = Math.min(text.length, matchIndex + term.length + context);
		const prefix = start > 0 ? '...' : '';
		const suffix = end < text.length ? '...' : '';

		return `${prefix}${text.slice(start, end).trim()}${suffix}`;
	}
</script>

{#if open}
	<IssueSearchDialog title={title ?? m['controls.issue_title']()} description={description ?? m['controls.issue_description']()} query={searchQuery} onquery={(value) => searchQuery = value}
		items={searchResults} onselect={selectIssue} loading={searching} {error} disabled={selecting} onretry={load} onclose={() => open = false}
		returnFocus={() => { if (returnFocus) returnFocus(); else origin?.focus(); origin = null; }} resultsLabel={!canSearchIssues(searchQuery) ? m['controls.issue_recent']() : undefined}>
		{#snippet metadata(id)}
			{@const result = searchResults.find(issue => issue.id === id)!}
			{@const snippet = descriptionSnippet(result.description, searchQuery)}
			<span class="mt-1 flex items-center gap-2 text-xs text-muted-foreground"><IssuePriorityIcon priority={result.priority} size={14} /><IssueStatusIcon status={result.status} category={result.status_info?.category} color={result.status_info?.color} size={14} />{actionLabel ?? m['controls.issue_action']()}</span>
			{#if snippet}<span class="mt-1 line-clamp-2 text-xs text-muted-foreground">{#each highlightedSegments(snippet, searchQuery) as segment}<span class={highlightClass(segment.match)}>{segment.text}</span>{/each}</span>{/if}
		{/snippet}
	</IssueSearchDialog>
{/if}
