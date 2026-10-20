import { expect, test, type Page, type Route } from '@playwright/test';
import type { TimesheetMode, TimesheetOptions, TimesheetGrid, TimesheetGroup, TimesheetWorklogs } from '../../src/lib/types/timesheets';

const a = '00000000-0000-0000-0000-000000000001';
const b = '00000000-0000-0000-0000-000000000002';
const userId = '00000000-0000-0000-0000-000000000010';
const otherId = '00000000-0000-0000-0000-000000000020';
const ghostId = '00000000-0000-0000-0000-000000000030';
const person = (id: string) => ({ id, available: id !== ghostId, name: id === ghostId ? null : 'Test User', display_name: null });
const baseWorkspaces = [
	{ id: a, name: 'Workspace A', slug: 'a', efforts_enabled: false, timezone: 'Europe/Rome', today: '2026-10-20', can_administer: true, can_record_work: false },
	{ id: b, name: 'Workspace B', slug: 'b', efforts_enabled: true, timezone: 'Pacific/Kiritimati', today: '2026-10-21', can_administer: false, can_record_work: true }
];
const selector = (page: Page, label: string) => page.getByRole('button', { name: label, exact: true }).and(page.locator('[data-slot="select-trigger"]'));
const scope = (page: Page) => page.getByRole('region', { name: 'Effective scope' });

async function fixture(page: Page, admin = true, theme: 'light' | 'dark' = 'dark', referenceToday = '2026-10-20', extraWorkspaceCount = 0, adminPeopleCount = 0) {
	const workspaces = [...baseWorkspaces, ...Array.from({ length: extraWorkspaceCount }, (_, i) => ({
		...baseWorkspaces[0], id: `00000000-0000-0000-0001-${String(i + 1).padStart(12, '0')}`, name: `Extra ${i}`, slug: `extra-${i}`, can_administer: false
	}))];
	const teams = workspaces.map((w, index) => ({ id: `team-${index}`, name: 'Same team', key: `T${index}`, workspace_id: w.id, icon: null, color: null }));
	const projects = workspaces.map((w, index) => ({ id: `project-${index}`, name: 'Same project', workspace_id: w.id }));

	const state = {
		admin, allowed: workspaces.map(w => w.id), fail: false, requests: [] as URL[],
		delay: null as null | { entered: () => void; wait: Promise<void> },
		dataDelay: null as null | { entered: () => void; wait: Promise<void> },
		detailDelay: null as null | { entered: () => void; wait: Promise<void> },
		gridRequests: [] as URL[], detailRequests: [] as URL[],
		dataFail: false, detailFail: false, emptyPeriod: false
	};
	const logDate = '2026-10-20';
	const records = [
		...Array.from({ length: 2 }, (_, i) => ({ id: `a-${i}`, workspaceId: a, issue: 'A-1', issueId: 'issue-a', project: projects[0].id, minutes: i === 0 ? 60 : 30 })),
		{ id: 'a-2', workspaceId: a, issue: 'A-2', issueId: 'issue-a2', project: null, minutes: 30 },
		...Array.from({ length: 51 }, (_, i) => ({ id: `b-${i}`, workspaceId: b, issue: 'B-1', issueId: 'issue-b', project: projects[1].id, minutes: 1 })),
		...workspaces.slice(2).map((w, i) => ({ id: `extra-${i}`, workspaceId: w.id, issue: `E-${i}`, issueId: `issue-extra-${i}`, project: projects[i + 2].id, minutes: 1 }))
	].map(record => ({ ...record, ownerId: userId }));
	const adminRecords = [...records,
		{ ...records[0], id: 'other', ownerId: otherId, minutes: 120 },
		{ ...records[2], id: 'ghost', ownerId: ghostId, minutes: 5 },
		...Array.from({ length: adminPeopleCount }, (_, i) => ({ ...records[0], id: `person-${i}`, ownerId: `00000000-0000-0000-0002-${String(i + 1).padStart(12, '0')}`, minutes: 1 }))
	];
	const selected = (url: URL, key: string, values: string[]) => {
		const raw = url.searchParams.get(key);
		return raw === null || raw === 'all' ? values : raw === '' ? [] : raw.split(',');
	};
	async function readTimesheet(route: Route, url: URL) {
		const detail = url.pathname.endsWith('/worklogs');
		(detail ? state.detailRequests : state.gridRequests).push(url);
		if (state.dataFail || (detail && state.detailFail)) return route.fulfill({ status: 500, json: { error: { code: 'INTERNAL_ERROR' } } });
		const mode = (url.searchParams.get('mode') ?? 'personal') as TimesheetMode;
		if (mode !== 'personal' && !state.admin) return route.fulfill({ status: 403, json: { error: { code: 'FORBIDDEN' } } });
		const memberships = workspaces.filter(w => state.allowed.includes(w.id)).map(w => ({ ...w, today: w.id === a ? referenceToday : w.today }));
		const admitted = memberships.filter(w => mode === 'personal' || w.can_administer);
		const workspaceIds = selected(url, 'workspace_ids', admitted.map(w => w.id));
		const reference = memberships.find(w => w.id === url.searchParams.get('reference_workspace_id'));
		if (!reference || workspaceIds.some(id => !admitted.some(w => w.id === id))) return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND' } } });
		if (detail && ((url.searchParams.get('issue_id') === 'issue-b' && !workspaceIds.includes(b)) || (url.searchParams.get('issue_id')?.startsWith('issue-a') && !workspaceIds.includes(a)))) return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND' } } });
		const teamIds = selected(url, 'team_ids', teams.filter(t => workspaceIds.includes(t.workspace_id)).map(t => t.id));
		const projectIds = selected(url, 'project_ids', [...projects.filter(p => workspaceIds.includes(p.workspace_id)).map(p => p.id), 'none']);
		const from = url.searchParams.get('from')!;
		const to = url.searchParams.get('to')!;
		const days: string[] = [];
		for (let day = new Date(`${from}T00:00:00Z`); day.toISOString().slice(0, 10) <= to; day.setUTCDate(day.getUTCDate() + 1)) days.push(day.toISOString().slice(0, 10));
		const inPeriod = !state.emptyPeriod && from <= logDate && logDate <= to;
		const candidates = (mode === 'personal' ? records : adminRecords).filter(r => inPeriod && workspaceIds.includes(r.workspaceId) && teamIds.includes(teams.find(t => t.workspace_id === r.workspaceId)!.id) && projectIds.includes(r.project ?? 'none'));
		const people = [...new Set(candidates.map(r => r.ownerId))].sort().map(person);
		const ownerIds = selected(url, 'owner_user_ids', people.map(p => p.id));
		const filtered = candidates.filter(r => ownerIds.includes(r.ownerId) && (!detail || !url.searchParams.has('issue_id') || r.issueId === url.searchParams.get('issue_id')) && (!detail || !url.searchParams.has('work_date') || url.searchParams.get('work_date') === logDate));
		const total = filtered.reduce((n, r) => n + r.minutes, 0);
		const daily = (minutes: number) => days.map(date => ({ date, minutes: date === logDate ? minutes : 0 }));
		const currentScope = { workspace_ids: workspaceIds, team_ids: teamIds, project_ids: projectIds.filter(id => id !== 'none'), include_no_project: projectIds.includes('none') };
		const pageNumber = Number(url.searchParams.get('page') ?? '1');
		const base = { mode, period: url.searchParams.get('period') as 'week' | 'month', from, to, days, reference, workspaces: admitted.filter(w => workspaceIds.includes(w.id)), scope: currentScope, total_minutes: total, daily_totals: daily(total), page: pageNumber, per_page: 50, total_count: 0, has_more: false };
		let response: TimesheetGrid | TimesheetWorklogs;
		if (detail) {
			response = { ...base, total_count: filtered.length, has_more: pageNumber * 50 < filtered.length, data: filtered.slice((pageNumber - 1) * 50, pageNumber * 50).map(r => ({
				id: r.id, issue_id: r.issueId, issue_identifier: r.issue, issue_title: `Work on ${r.issue}`, workspace: workspaces.find(w => w.id === r.workspaceId)!, team: teams.find(t => t.workspace_id === r.workspaceId)!,
				owner: person(r.ownerId), operator: { id: 'admin', available: true, name: 'Workspace Admin', display_name: null },
				work_date: logDate, duration_minutes: r.minutes, description: `Registration ${r.id}`, created_at: '2026-10-20T12:00:00Z', updated_at: '2026-10-20T12:00:00Z'
			})) };
		} else {
			const groups: TimesheetGroup[] = [];
			const nodes = new Map<string, TimesheetGroup>();
			for (const r of filtered) {
				const workspace = workspaces.find(w => w.id === r.workspaceId)!;
				const team = teams.find(t => t.workspace_id === r.workspaceId)!;
				const workspaceGroup = { key: `workspace:${workspace.id}`, type: 'workspace', id: workspace.id, name: workspace.name };
				const teamGroup = { key: `team:${team.id}`, type: 'team', id: team.id, name: team.name, team };
				const projectGroup = { key: `project:${workspace.id}:${r.project}`, type: r.project ? 'project' : 'no_project', id: r.project, name: r.project ? 'Same project' : '' };
				const personGroup = { key: `person:${r.ownerId}`, type: 'person', id: r.ownerId, name: person(r.ownerId).name ?? '', person: person(r.ownerId) };
				const issueGroup = { key: `issue:${r.issueId}`, type: 'issue', id: r.issueId, name: `Work on ${r.issue}`, identifier: r.issue };
				const path = mode === 'person' ? [personGroup,workspaceGroup,teamGroup,projectGroup,issueGroup]
					: mode === 'project' ? [projectGroup,teamGroup,personGroup,issueGroup]
					: mode === 'workspace' ? [workspaceGroup,teamGroup,projectGroup,personGroup,issueGroup]
					: mode === 'team' ? [teamGroup,projectGroup,personGroup,issueGroup]
					: [workspaceGroup,teamGroup,projectGroup,issueGroup];
				let parent: TimesheetGroup | undefined;
				for (const item of path) {
					const key = parent ? `${parent.key}/${item.key}` : item.key;
					let node = nodes.get(key);
					if (!node) {
						node = { ...item, key, type: item.type as TimesheetGroup['type'], workspace_id: !parent && mode === 'person' ? null : workspace.id, workspace_slug: !parent && mode === 'person' ? null : workspace.slug, children: [], daily_totals: daily(0), total_minutes: 0 };
						nodes.set(key, node);
						if (parent) parent.children.push(node); else groups.push(node);
					}
					node.total_minutes += r.minutes;
					node.daily_totals.find(day => day.date === logDate)!.minutes += r.minutes;
					parent = node;
				}
			}
			response = { ...base, people, total_count: groups.length, has_more: pageNumber * 50 < groups.length, groups: groups.slice((pageNumber - 1) * 50, pageNumber * 50), empty_reason: groups.length ? null : inPeriod ? 'filters' : 'period' };
		}
		const delay = detail ? state.detailDelay : state.dataDelay;
		if (delay) { if (detail) state.detailDelay = null; else state.dataDelay = null; delay.entered(); await delay.wait; }
		return route.fulfill({ json: response });
	}
	await page.addInitScript(() => localStorage.setItem('PARAGLIDE_LOCALE', 'en'));
	await page.routeWebSocket('**/ws', () => {});
	await page.route('https://raw.githubusercontent.com/carbogninalberto/kuayle/main/UI/static/releases.json', (route) => route.fulfill({ json: [] }));
	await page.route('**/api/**', async (route: Route) => {
		const url = new URL(route.request().url());
		const path = url.pathname;
		if (path === '/api/auth/me') return route.fulfill({ json: { id: userId, email: 'test@example.com', name: 'Test User', avatar_url: null } });
		if (path === '/api/preferences') return route.fulfill({ json: { theme_mode: theme, light_theme: 'light', dark_theme: 'dark', font_size: 'default', pointer_cursors: true } });
		if (path === '/api/workspaces') return route.fulfill({ json: workspaces.map((w) => ({ ...w, current_user_role: 'member' })) });
		if (path === '/api/workspaces/a' || path === '/api/workspaces/b') return route.fulfill({ json: { ...workspaces[path.endsWith('/b') ? 1 : 0], owner_id: 'other', current_user_role: 'member', share_link_min_role: 'admin' } });
		if (path === '/api/workspaces/a/efforts/settings') return route.fulfill({ json: { enabled: false, timezone: 'Europe/Rome', today: '2026-10-20', revision: 0 } });
		if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
		if (path === '/api/timesheets/options') {
			state.requests.push(url);
			if (state.fail) return route.fulfill({ status: 500, json: { error: { code: 'INTERNAL_ERROR' } } });
			const mode = (url.searchParams.get('mode') ?? 'personal') as TimesheetMode;
			if (mode !== 'personal' && !state.admin) return route.fulfill({ status: 403, json: { error: { code: 'FORBIDDEN' } } });
			const admitted = workspaces.filter((w) => state.allowed.includes(w.id) && (mode === 'personal' || (state.admin && w.can_administer))).map(w => ({ ...w, today: w.id === a ? referenceToday : w.today }));
			const selected = (key: string, values: string[]) => {
				const raw = url.searchParams.get(key);
				return raw === null || raw === 'all' ? values : raw === '' ? [] : raw.split(',');
			};
			const workspaceIds = selected('workspace_ids', admitted.map((w) => w.id));
			const teamOptions = teams.filter((t) => workspaceIds.includes(t.workspace_id));
			const projectOptions = projects.filter((p) => workspaceIds.includes(p.workspace_id));
			const teamIds = selected('team_ids', teamOptions.map((t) => t.id));
			const projectIds = selected('project_ids', [...projectOptions.map((p) => p.id), 'none']);
			if (workspaceIds.some((id) => !admitted.some((w) => w.id === id)) || teamIds.some((id) => !teamOptions.some((t) => t.id === id)) || projectIds.some((id) => id !== 'none' && !projectOptions.some((p) => p.id === id))) {
				return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND' } } });
			}
			const response: TimesheetOptions = {
				mode, available_modes: state.admin ? ['personal', 'person', 'project', 'workspace', 'team'] : ['personal'],
				workspaces: admitted.map((w) => ({ ...w, can_administer: state.admin && w.can_administer })), teams: teamOptions, projects: projectOptions,
				scope: { workspace_ids: workspaceIds, team_ids: teamIds, project_ids: projectIds.filter((id) => id !== 'none'), include_no_project: projectIds.includes('none') }
			};
			if (state.delay) {
				const delay = state.delay;
				state.delay = null;
				delay.entered();
				await delay.wait;
			}
			return route.fulfill({ json: response });
		}
		if (path === '/api/timesheets' || path === '/api/timesheets/worklogs') return readTimesheet(route, url);
		if (/\/issues\/[^/]+$/.test(path)) return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND' } } });
		if (path.endsWith('/issues')) return route.fulfill({ json: { data: [], total_count: 0, page: 1, per_page: 50, has_more: false } });
		return route.fulfill({ json: [] });
	});
	await page.goto('/a/timesheet');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await expect(page.locator('html')).toHaveClass(new RegExp(`\\b${theme}\\b`));
	return state;
}

async function choose(page: Page, label: string, name: string) {
	await selector(page, label).click();
	await page.getByRole('option', { name, exact: true }).click();
	await page.keyboard.press('Escape');
	await expect(page.locator('[data-slot="select-content"]')).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
}

test('sidebar, initial cross-workspace scope and local capabilities', async ({ page }, testInfo) => {
	const state = await fixture(page);
	await expect(page.getByRole('link', { name: 'Timesheet', exact: true })).toHaveAttribute('href', '/a/timesheet');
	await expect(scope(page)).toContainText('Workspace A (a)');
	await expect(scope(page)).toContainText('Workspace B (b)');
	await expect(scope(page)).toContainText('Read only');
	await expect(scope(page)).toContainText('Recording allowed');
	await expect(page.getByRole('table', { name: 'Timesheet grid' })).toBeVisible();
	expect(state.requests[0].searchParams.get('workspace_ids')).toBe('all');
	await page.screenshot({ path: testInfo.outputPath('timesheet-desktop.png'), fullPage: true });
	await choose(page, 'View', 'By person');
	await expect(scope(page)).toContainText('Workspace A (a)');
	await expect(scope(page)).not.toContainText('Workspace B (b)');
	await expect(page.getByRole('status')).toContainText('scope was updated');
	await choose(page, 'View', 'My timesheet');
	await expect(scope(page)).not.toContainText('Workspace B (b)');
});

test('multiple filters retain namesakes and remove only incompatible children', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'Teams', 'Same team · Workspace A (a)');
	await choose(page, 'Teams', 'Same team · Workspace B (b)');
	await choose(page, 'Projects', 'Same project · Workspace B (b)');
	await choose(page, 'Projects', 'No project');
	await choose(page, 'Workspaces', 'Workspace A (a)');
	const last = state.requests.at(-1)!;
	expect(last.searchParams.get('team_ids')).toBe('team-0');
	expect(last.searchParams.get('project_ids')).toBe('none');
	await expect(page.getByRole('status')).toContainText('Filters outside');
	await choose(page, 'Workspaces', 'Workspace A (a)');
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	expect(state.requests.at(-1)!.searchParams.get('workspace_ids')).toBe('');
});

test('member/guest modes and empty scopes stay explicit', async ({ page }) => {
	const state = await fixture(page, false);
	await selector(page, 'View').click();
	await expect(page.getByRole('option', { name: 'By person', exact: true })).toHaveCount(0);
	await page.keyboard.press('Escape');
	await choose(page, 'Workspaces', 'All');
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	state.allowed = [];
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByText('No workspaces are available.')).toBeVisible();
});

test('revoked IDs are removed before rereading, without broadening an empty selection', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'Workspaces', 'Workspace B (b)');
	await choose(page, 'Teams', 'Same team · Workspace B (b)');
	state.allowed = [a];
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByRole('status')).toContainText('current access');
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	await expect(scope(page)).toHaveCount(0);
	expect(state.requests.at(-1)!.searchParams.get('workspace_ids')).toBe('');
	expect(state.requests.at(-1)!.searchParams.get('team_ids')).toBe('');
});

test('a downgrade removes administrative modes and preserves the selected workspace', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'View', 'By team');
	state.admin = false;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(selector(page, 'View')).toContainText('My timesheet');
	await expect(scope(page)).toContainText('Workspace A (a)');
	await expect(scope(page)).not.toContainText('Workspace B (b)');
	await expect(page.getByRole('status')).toContainText('current access');
});

test('failed reads show an error and retry; old responses cannot restore the old scope', async ({ page }) => {
	const state = await fixture(page);
	state.fail = true;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByText('Unable to load Timesheet.')).toBeVisible();
	await expect(scope(page)).toHaveCount(0);
	state.fail = false;
	await page.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(scope(page)).toBeVisible();
	let release!: () => void;
	let entered!: () => void;
	const waiting = new Promise<void>((resolve) => { entered = resolve; });
	state.delay = { entered, wait: new Promise<void>((resolve) => { release = resolve; }) };
	await selector(page, 'Workspaces').click();
	await page.getByRole('option', { name: 'Workspace A (a)', exact: true }).click();
	await page.keyboard.press('Escape');
	await waiting;
	await choose(page, 'Workspaces', 'Workspace B (b)');
	// Multiple selection now contains A+B. The delayed A-only response must not win.
	await expect(scope(page)).toContainText('Workspace B (b)');
	const staleResponse = page.waitForResponse((response) => new URL(response.url()).searchParams.get('workspace_ids') === a);
	release();
	await staleResponse;
	await page.evaluate(() => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
	await expect.poll(() => selector(page, 'Workspaces').textContent()).toContain('2 selected');
	await expect(scope(page)).toContainText('Workspace B (b)');
});

test('filters work with keyboard and fit a narrow viewport', async ({ page }, testInfo) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await fixture(page, true, 'light');
	const teams = selector(page, 'Teams');
	await teams.focus();
	await page.keyboard.press('Enter');
	await expect(page.getByRole('option', { name: 'Same team · Workspace A (a)', exact: true })).toBeVisible();
	await page.keyboard.press('ArrowDown');
	await page.keyboard.press('Enter');
	await page.keyboard.press('Escape');
	await expect(teams).toBeFocused();
	await expect(page.locator('[data-slot="select-content"]')).toHaveCount(0);
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
	await page.screenshot({ path: testInfo.outputPath('timesheet-mobile.png'), fullPage: true });
});

test('personal grid keeps complete totals, hierarchy and distinct registrations', async ({ page }) => {
	await fixture(page);
	const table = page.getByRole('table', { name: 'Timesheet grid' });
	await expect(table.locator('thead th')).toHaveCount(9);
	await expect(table.locator('tfoot')).toContainText('2h 51m');
	await expect(table.getByRole('link', { name: 'A-2 Work on A-2' })).toBeVisible();
	await table.getByRole('button', { name: 'Workspace A', exact: true }).click();
	await expect(table.getByRole('button', { name: 'A-1 · 2026-10-20 · 1h 30m', exact: true })).toHaveCount(0);
	await expect(table.locator('tfoot')).toContainText('2h 51m');
	await table.getByRole('button', { name: 'Workspace A', exact: true }).click();
	const cell = table.getByRole('button', { name: 'A-1 · 2026-10-20 · 1h 30m', exact: true });
	await cell.focus(); await page.keyboard.press('Enter');
	const dialog = page.getByRole('dialog', { name: 'Worklog details' });
	await expect(dialog.getByText('2 registrations')).toBeVisible();
	await expect(dialog.locator('li')).toHaveCount(2);
	await expect(dialog).toContainText('Workspace Admin');
	await expect(dialog).toContainText('Test User');
	await expect(dialog).toContainText('Europe/Rome');
	await expect(dialog.getByRole('link').first()).toHaveAttribute('href', '/a/issue/A-1');
	await page.keyboard.press('Escape');
	await expect(dialog).toHaveCount(0);
	await expect(cell).toBeFocused();
});

test('week/month navigation uses the reference and retains dates across filters', async ({ page }) => {
	const state = await fixture(page);
	const initial = state.gridRequests[0];
	expect(initial.searchParams.get('from')).toBe('2026-10-19');
	expect(initial.searchParams.get('to')).toBe('2026-10-25');
	await choose(page, 'Workspaces', 'Workspace B (b)');
	expect(state.gridRequests.at(-1)!.searchParams.get('reference_workspace_id')).toBe(a);
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-19');
	await page.getByRole('button', { name: 'Month', exact: true }).click();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-01');
	await page.getByRole('button', { name: 'Previous period', exact: true }).click();
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(32);
	await expect(page.getByRole('table').locator('tbody tr')).toHaveCount(0);
	expect(state.gridRequests.at(-1)!.searchParams.get('to')).toBe('2026-09-30');
	await page.getByRole('button', { name: 'Current period', exact: true }).click();
	await expect(page.getByRole('table')).toBeVisible();
	await page.getByRole('button', { name: 'Week', exact: true }).click();
	await page.getByRole('button', { name: 'Current period', exact: true }).click();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(9);
	await page.getByRole('button', { name: 'Select period', exact: true }).click();
	await page.locator('[data-bits-day]').filter({ hasText: /^15$/ }).click();
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-12');
});

test('all detail pages remain reachable and preserve full totals on mobile', async ({ page }, testInfo) => {
	await page.setViewportSize({ width: 390, height: 844 });
	const state = await fixture(page, false, 'light');
	await page.getByRole('button', { name: 'Month', exact: true }).click();
	const region = page.getByRole('region', { name: 'Timesheet grid' });
	expect(await region.evaluate(el => el.scrollWidth > el.clientWidth)).toBe(true);
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Worklog details' });
	await expect(dialog.locator('li')).toHaveCount(50);
	await expect(dialog.getByRole('status')).toContainText('51m');
	await dialog.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(dialog.locator('li')).toHaveCount(1);
	await expect(dialog).toContainText('Registration b-50');
	await expect(dialog.getByRole('status')).toContainText('51 registrations');
	expect(state.detailRequests.at(-1)!.searchParams.get('page')).toBe('2');
	await page.screenshot({ path: testInfo.outputPath('timesheet-detail-mobile.png'), fullPage: true });
	await dialog.getByRole('button', { name: 'Close', exact: true }).last().click();
});

test('returning from the correct workspace issue restores session filters and period', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'Workspaces', 'Workspace B (b)');
	await page.getByRole('button', { name: 'Month', exact: true }).click();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true }).click();
	await page.getByRole('dialog').getByRole('link').first().click();
	await expect(page).toHaveURL(/\/b\/issue\/B-1$/);
	await page.goBack();
	await expect(page).toHaveURL(/\/a\/timesheet$/);
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
	await expect(scope(page)).not.toContainText('Workspace A (a)');
	expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe(b);
	expect(state.gridRequests.at(-1)!.searchParams.get('reference_workspace_id')).toBe(a);
});

test('reference revocation retains the selected period and removes old data', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'Month', exact: true }).click();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
	state.allowed = [b];
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByRole('status')).toContainText('reference workspace changed');
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-01');
	expect(state.gridRequests.at(-1)!.searchParams.get('reference_workspace_id')).toBe(b);
	await expect(page.getByRole('table')).not.toContainText('A-1');
});

test('grid and detail errors never appear as zero, and empty states remain distinct', async ({ page }) => {
	const state = await fixture(page);
	state.dataFail = true;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByText('Unable to load Timesheet.')).toBeVisible();
	await expect(page.getByRole('table')).toHaveCount(0);
	state.dataFail = false;
	await page.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(page.getByRole('table')).toBeVisible();
	state.detailFail = true;
	await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1h 30m', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByText('Unable to load Timesheet.')).toBeVisible();
	state.detailFail = false;
	await dialog.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(dialog.locator('li')).toHaveCount(2);
	await page.keyboard.press('Escape');
	await choose(page, 'Projects', 'All');
	await expect(page.getByText('No work matches the selected filters.')).toBeVisible();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(9);
	await choose(page, 'Projects', 'All');
	state.emptyPeriod = true;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
});

test('late grid/detail responses cannot restore an obsolete selection or dialog', async ({ page }) => {
	const state = await fixture(page);
	let release!: () => void;
	let entered!: () => void;
	const waiting = new Promise<void>(resolve => { entered = resolve; });
	state.dataDelay = { entered, wait: new Promise<void>(resolve => { release = resolve; }) };
	await selector(page, 'Workspaces').click();
	await page.getByRole('option', { name: 'Workspace A (a)', exact: true }).click();
	await page.keyboard.press('Escape');
	await waiting;
	await choose(page, 'Workspaces', 'Workspace B (b)');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('2h 51m');
	const stale = page.waitForResponse(res => new URL(res.url()).pathname === '/api/timesheets' && new URL(res.url()).searchParams.get('workspace_ids') === a);
	release(); await stale;
	await expect(page.getByRole('table').locator('tfoot')).toContainText('2h 51m');
	let releaseDetail!: () => void;
	let enteredDetail!: () => void;
	const waitingDetail = new Promise<void>(resolve => { enteredDetail = resolve; });
	state.detailDelay = { entered: enteredDetail, wait: new Promise<void>(resolve => { releaseDetail = resolve; }) };
	await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1h 30m', exact: true }).click();
	await waitingDetail;
	await page.keyboard.press('Escape');
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true }).click();
	await expect(page.getByRole('dialog')).toContainText('Registration b-0');
	const staleDetail = page.waitForResponse(res => new URL(res.url()).pathname === '/api/timesheets/worklogs' && new URL(res.url()).searchParams.get('issue_id') === 'issue-a');
	releaseDetail(); await staleDetail;
	await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
	await expect(page.getByRole('dialog')).not.toContainText('Registration a-0');
});

test('workspace pagination retains full totals', async ({ page }) => {
	const state = await fixture(page, true, 'dark', '2026-10-20', 50);
	await page.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(page.getByText('Page 2 of 2', { exact: true })).toBeVisible();
	await expect(page.getByRole('table').locator('tfoot')).toContainText('3h 41m');
	expect(state.gridRequests.at(-1)!.searchParams.get('page')).toBe('2');
	await page.getByRole('button', { name: 'Previous', exact: true }).click();
	await expect(page.getByText('Page 1 of 2', { exact: true })).toBeVisible();
});

test('the initial/current week comes from server today even across the Sunday boundary', async ({ page }) => {
	const state = await fixture(page, false, 'dark', '2026-10-26');
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	expect(state.gridRequests[0].searchParams.get('from')).toBe('2026-10-26');
	await page.getByRole('button', { name: 'Previous period', exact: true }).click();
	await expect(page.getByRole('table')).toBeVisible();
	await page.getByRole('button', { name: 'Current period', exact: true }).click();
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-26');
});

test('lost access while paging detail removes the dialog and revoked workspace', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Worklog details' });
	await expect(dialog.locator('li')).toHaveCount(50);
	state.allowed = [a];
	await dialog.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(dialog).toHaveCount(0);
	await expect(page.getByRole('status')).toContainText('current access');
	await expect(page.getByRole('table')).not.toContainText('B-1');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('2h');
});

test('all administrative hierarchies preserve totals, scope and current classification', async ({ page }, testInfo) => {
 const state = await fixture(page);
 for (const label of ['By person','By project','By workspace','By team']) {
  await choose(page,'View',label);
  await expect(page.getByRole('table').locator('tfoot')).toContainText('4h 5m');
  await expect(scope(page)).not.toContainText('Workspace B (b)');
  await expect(page.getByText('Account unavailable · 00000030',{exact:true})).toBeVisible();
  await expect(page.getByRole('button',{name:'A-1 · 2026-10-20 · 1h 30m',exact:true})).toBeVisible();
  await expect(page.getByRole('button',{name:'A-1 · 2026-10-20 · 2h',exact:true})).toBeVisible();
  expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe('all');
 }
 await page.screenshot({path:testInfo.outputPath('timesheet-admin-desktop-dark.png'),fullPage:true});
});

test('administrative people filters combine, preserve empty selections and survive mode switches', async ({ page }) => {
 const state = await fixture(page);
 await choose(page,'View','By person');
 await choose(page,'People','Test User · 00000010');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('2h');
 await choose(page,'People','Test User · 00000020');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('4h');
 await choose(page,'Projects','No project');
 for (const label of ['By project','By workspace','By team','By person']) {
  await choose(page,'View',label);
  await expect(page.getByRole('table').locator('tfoot')).toContainText('30m');
  expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(`${userId},${otherId}`);
  expect(state.gridRequests.at(-1)!.searchParams.get('project_ids')).toBe('none');
 }
 await choose(page,'People','Test User · 00000010');
 await choose(page,'People','Account unavailable · 00000020');
 await expect(page.getByText('No work matches the selected filters.')).toBeVisible();
 expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe('');
 await expect(selector(page,'People')).toContainText('None selected');
});

test('opening a person focuses the same report and cell detail includes only that owner', async ({ page }) => {
 const state = await fixture(page);
 await choose(page,'View','By person');
 await page.getByRole('button',{name:'Open timesheet · Test User · 00000020',exact:true}).click();
 await expect(selector(page,'View')).toContainText('By person');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('2h');
 expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(otherId);
 expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe(a);
 await choose(page,'People','All');
 await choose(page,'View','By project');
 const cell = page.getByRole('button',{name:'A-1 · 2026-10-20 · 2h',exact:true});
 await cell.focus();await page.keyboard.press('Enter');
 const dialog=page.getByRole('dialog');
 await expect(dialog.getByRole('status')).toContainText('2h · 1 registrations');
 expect(state.detailRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(otherId);
 expect(state.detailRequests.at(-1)!.searchParams.get('mode')).toBe('project');
 await expect(dialog).toContainText('Registration other');
 await expect(dialog).not.toContainText('Registration a-0');
 await page.keyboard.press('Escape');await expect(cell).toBeFocused();
});

test('administrative history restores people, period and grouping after visiting an issue', async ({ page }) => {
 const state=await fixture(page);
 await choose(page,'View','By person');
 await choose(page,'People','Test User · 00000020');
 await page.getByRole('button',{name:'Month',exact:true}).click();
 await page.getByRole('button',{name:'A-1 · 2026-10-20 · 2h',exact:true}).click();
 await page.getByRole('dialog').getByRole('link').first().click();
 await expect(page).toHaveURL(/\/a\/issue\/A-1$/);
 await page.goBack();
 await expect(selector(page,'View')).toContainText('By person');
 await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
 await expect(page.getByRole('table').locator('tfoot')).toContainText('2h');
 expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(otherId);
});

test('administrative pagination retains complete totals and all person options', async ({ page }) => {
 const state=await fixture(page,true,'dark','2026-10-20',0,51);
 await choose(page,'View','By person');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('4h 56m');
 await page.getByRole('button',{name:'Next',exact:true}).last().click();
 await expect(page.getByText('Page 2 of 2')).toBeVisible();
 await expect(page.getByRole('table').locator('tfoot')).toContainText('4h 56m');
 expect(state.gridRequests.at(-1)!.searchParams.get('page')).toBe('2');
 await selector(page,'People').click();
 await expect(page.getByRole('option')).toHaveCount(55);
 await page.keyboard.press('Escape');
});

test('unavailable identities and administrative detail remain usable on mobile in light theme', async ({ page },testInfo) => {
 await page.setViewportSize({width:390,height:844});
 const state=await fixture(page,true,'light');
 await choose(page,'View','By person');
 await choose(page,'People','Account unavailable · 00000030');
 await page.getByRole('button',{name:'A-2 · 2026-10-20 · 5m',exact:true}).click();
 const dialog=page.getByRole('dialog');
 await expect(dialog).toContainText('Account unavailable');
 expect(state.detailRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(ghostId);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
 await page.screenshot({path:testInfo.outputPath('timesheet-admin-mobile-light.png'),fullPage:true});
});

test('administrative failures and downgrades remove totals and recover within the current scope', async ({ page }) => {
 const state=await fixture(page);
 await choose(page,'View','By person');
 state.dataFail=true;
 await page.getByRole('button',{name:'Refresh',exact:true}).click();
 await expect(page.getByText('Unable to load Timesheet.')).toBeVisible();
 await expect(page.getByRole('table')).toHaveCount(0);
 state.dataFail=false;
 await page.getByRole('button',{name:'Try again',exact:true}).click();
 await expect(page.getByRole('table').locator('tfoot')).toContainText('4h 5m');
 state.admin=false;
 await page.getByRole('button',{name:'A-1 · 2026-10-20 · 2h',exact:true}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(selector(page,'View')).toContainText('My timesheet');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('2h');
 await expect(scope(page)).not.toContainText('Workspace B (b)');
});

test('late administrative responses cannot replace a newer project filter', async ({ page }) => {
 const state=await fixture(page);
 await choose(page,'View','By person');
 let release!:()=>void;let entered!:()=>void;
 const waiting=new Promise<void>(resolve=>{entered=resolve;});
 state.dataDelay={entered,wait:new Promise<void>(resolve=>{release=resolve;})};
 await page.getByRole('button',{name:'Refresh',exact:true}).click();
 await waiting;
 await choose(page,'Projects','No project');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('35m');
 const stale=page.waitForResponse(response=>{const url=new URL(response.url());return url.pathname==='/api/timesheets'&&url.searchParams.get('project_ids')==='all';});
 release();await stale;
 await expect(page.getByRole('table').locator('tfoot')).toContainText('35m');
});

test('an administrative report retains a personal reference outside its data scope', async ({ page }) => {
 const state=await fixture(page);
 await page.goto('/b/timesheet');
 await expect(page.getByRole('button',{name:'Refresh',exact:true})).toBeEnabled();
 await choose(page,'View','By person');
 await expect(page.getByText('Reference: Workspace B (b)',{exact:false})).toBeVisible();
 expect(state.gridRequests.at(-1)!.searchParams.get('reference_workspace_id')).toBe(b);
 expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe(a);
 await expect(page.getByRole('table').locator('tfoot')).toContainText('4h 5m');
});
