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

async function fixture(page: Page, admin = true, theme: 'light' | 'dark' = 'dark', referenceToday = '2026-10-20', extraWorkspaceCount = 0, adminPeopleCount = 0, entry = '/a/timesheet', locale = 'en') {
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
		dataFail: false, detailFail: false, emptyPeriod: false, invalidRead: false, longDetailText: false
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
		if (state.invalidRead) return route.fulfill({ status: 400, json: { error: { code: 'INVALID_INPUT' } } });
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
				id: r.id, issue_id: r.issueId, issue_identifier: r.issue, issue_title: state.longDetailText ? 'LongTitle'.repeat(35) : `Work on ${r.issue}`, workspace: workspaces.find(w => w.id === r.workspaceId)!, team: teams.find(t => t.workspace_id === r.workspaceId)!,
				owner: state.longDetailText ? { ...person(r.ownerId), name: 'LongName'.repeat(25) } : person(r.ownerId), operator: { id: 'admin', available: true, name: 'Workspace Admin', display_name: null },
				work_date: logDate, duration_minutes: r.minutes, description: state.longDetailText ? 'LongDescription'.repeat(60) : `Registration ${r.id}`, created_at: '2026-10-20T12:00:00Z', updated_at: '2026-10-20T12:00:00Z'
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
	await page.addInitScript((locale) => localStorage.setItem('PARAGLIDE_LOCALE', locale), locale);
	await page.routeWebSocket(/\/ws(?:\?|$)/, () => {});
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
	await page.goto(entry);
	await expect(page.getByRole('button', { name: ({ en: 'Refresh', it: 'Aggiorna', 'zh-CN': '刷新', 'zh-TW': '重新整理' }[locale] ?? 'Refresh'), exact: true })).toBeEnabled();
	await expect(page.locator('html')).toHaveClass(new RegExp(`\\b${theme}\\b`));
	return state;
}

async function choose(page: Page, label: string, name: string) {
	if (label === 'View') {
		await page.getByRole('tab', { name, exact: true }).click();
		await expect(page.getByRole('tab', { name, exact: true })).toHaveAttribute('aria-selected', 'true');
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		return;
	}
	const filterButton = page.getByRole('button', { name: 'Filters', exact: true });
	if (await filterButton.isVisible() && await filterButton.getAttribute('aria-expanded') === 'false') await filterButton.click();
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
	await expect(page.getByRole('status').filter({ hasText: 'scope was updated' })).toContainText('scope was updated');
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
	await expect(page.getByRole('status').filter({ hasText: 'Filters outside' })).toContainText('Filters outside');
	await choose(page, 'Workspaces', 'Workspace A (a)');
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	expect(state.requests.at(-1)!.searchParams.get('workspace_ids')).toBe('');
});

test('member/guest modes and empty scopes stay explicit', async ({ page }) => {
	const state = await fixture(page, false);
	await expect(page.getByRole('tab')).toHaveCount(1);
	await expect(page.getByRole('tab', { name: 'By person', exact: true })).toHaveCount(0);
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
	await expect(page.getByRole('status').filter({ hasText: 'current access' })).toContainText('current access');
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
	await expect(page.getByRole('tab', { selected: true })).toContainText('My timesheet');
	await expect(scope(page)).toContainText('Workspace A (a)');
	await expect(scope(page)).not.toContainText('Workspace B (b)');
	await expect(page.getByRole('status').filter({ hasText: 'current access' })).toContainText('current access');
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
	await expect(selector(page, 'Workspaces')).toHaveAttribute('aria-expanded', 'false');
	await waiting;
	await choose(page, 'Workspaces', 'Workspace B (b)');
	// Multiple selection now contains A+B. The delayed A-only response must not win.
	await expect(scope(page)).toContainText('Workspace B (b)');
	const currentUrl = page.url();
	const staleResponse = page.waitForResponse((response) => new URL(response.url()).searchParams.get('workspace_ids') === a);
	release();
	await staleResponse;
	await page.evaluate(() => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
	await expect(page).toHaveURL(currentUrl);
	await expect.poll(() => selector(page, 'Workspaces').textContent()).toContain('Workspace A (a) +1');
	await expect(scope(page)).toContainText('Workspace B (b)');
});

test('filters work with keyboard and fit a narrow viewport', async ({ page }, testInfo) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await fixture(page, true, 'light');
	await page.getByRole('button', { name: 'Filters', exact: true }).click();
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
	await expect(scope(page)).toContainText('Filtered period total: 0.36d');
	await expect(table.getByRole('row').filter({ has: page.getByRole('button', { name: 'Workspace A', exact: true }) }).locator('td').last()).toHaveText('0.25d');
	const issueRow = table.getByRole('row').filter({ has: page.getByRole('link', { name: 'A-1 Work on A-1', exact: true }) });
	await expect(issueRow.locator('td').last()).toHaveText('0.19d');
	await expect(issueRow.getByRole('button', { name: 'A-1 · 2026-10-20 · 1.50h', exact: true })).toBeVisible();
	await expect(table.locator('tfoot td')).toHaveText(['0.00d', '0.36d', '0.00d', '0.00d', '0.00d', '0.00d', '0.00d', '0.36d']);
	await expect(table.locator('tfoot')).toContainText('0.36d');
	await expect(table.getByRole('link', { name: 'A-2 Work on A-2' })).toBeVisible();
	await table.getByRole('button', { name: 'Workspace A', exact: true }).click();
	await expect(table.getByRole('button', { name: 'A-1 · 2026-10-20 · 1.50h', exact: true })).toHaveCount(0);
	await expect(table.locator('tfoot')).toContainText('0.36d');
	await table.getByRole('button', { name: 'Workspace A', exact: true }).click();
	const cell = table.getByRole('button', { name: 'A-1 · 2026-10-20 · 1.50h', exact: true });
	await cell.focus(); await page.keyboard.press('Enter');
	const dialog = page.getByRole('dialog', { name: 'Worklog details' });
	await expect(dialog.getByText('2 registrations')).toBeVisible();
	await expect(dialog.locator('li')).toHaveCount(2);
	await expect(dialog.getByRole('status')).toContainText('Total: 0.19d');
	await expect(dialog.locator('li strong')).toHaveText(['1.00h', '0.50h']);
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
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 0.85h', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Worklog details' });
	await expect(dialog.locator('li')).toHaveCount(50);
	await expect(dialog.getByRole('status')).toContainText('0.11d');
	await dialog.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(dialog.locator('li')).toHaveCount(1);
	await expect(dialog).toContainText('Registration b-50');
	await expect(dialog.getByRole('status')).toContainText('51 registrations');
	expect(state.detailRequests.at(-1)!.searchParams.get('page')).toBe('2');
	await page.screenshot({ path: testInfo.outputPath('timesheet-detail-mobile.png'), fullPage: true });
	await dialog.getByRole('button', { name: 'Close', exact: true }).last().click();
});

test('returning from the correct workspace issue restores URL filters and period', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'Workspaces', 'Workspace B (b)');
	await page.getByRole('button', { name: 'Month', exact: true }).click();
	await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 0.85h', exact: true }).click();
	await page.getByRole('dialog').getByRole('link').first().click();
	await expect(page).toHaveURL(/\/b\/issue\/B-1$/);
	await page.goBack();
	await expect(page).toHaveURL(/\/a\/timesheet\?/);
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
	await expect(page.getByRole('status').filter({ hasText: 'reference workspace changed' })).toContainText('reference workspace changed');
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
	await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1.50h', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByText('Unable to load Timesheet.')).toBeVisible();
	state.detailFail = false;
	await dialog.getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(dialog.locator('li')).toHaveCount(2);
	await expect(dialog.getByRole('status')).toContainText('Total: 0.19d');
	await expect(dialog.locator('li strong')).toHaveText(['1.00h', '0.50h']);
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
	await expect(selector(page, 'Workspaces')).toHaveAttribute('aria-expanded', 'false');
	await waiting;
	await choose(page, 'Workspaces', 'Workspace B (b)');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.36d');
	const currentUrl = page.url();
	const stale = page.waitForResponse(res => new URL(res.url()).pathname === '/api/timesheets' && new URL(res.url()).searchParams.get('workspace_ids') === a);
	release(); await stale;
	await expect(page).toHaveURL(currentUrl);
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.36d');
	let releaseDetail!: () => void;
	let enteredDetail!: () => void;
	const waitingDetail = new Promise<void>(resolve => { enteredDetail = resolve; });
	state.detailDelay = { entered: enteredDetail, wait: new Promise<void>(resolve => { releaseDetail = resolve; }) };
	await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1.50h', exact: true }).click();
	await waitingDetail;
	await page.keyboard.press('Escape');
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 0.85h', exact: true }).click();
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
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.46d');
	expect(state.gridRequests.at(-1)!.searchParams.get('page')).toBe('2');
	await page.getByRole('button', { name: 'Previous', exact: true }).click();
	await expect(page.getByText('Page 1 of 2', { exact: true })).toBeVisible();
});

test('the initial/current week comes from server today even across the Sunday boundary', async ({ page }) => {
	const state = await fixture(page, false, 'dark', '2026-10-26');
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	expect(state.gridRequests[0].searchParams.get('from')).toBe('2026-10-26');
	await page.getByRole('button', { name: 'Select period' }).click();
	await expect(page.locator('[data-bits-day][data-today]')).toHaveText('26');
	await expect(page.locator('[data-bits-day][aria-current="date"]')).toHaveText('26');
	await page.keyboard.press('Escape');
	await page.getByRole('button', { name: 'Previous period', exact: true }).click();
	await expect(page.getByRole('table')).toBeVisible();
	await page.getByRole('button', { name: 'Current period', exact: true }).click();
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-26');
});

test('lost access while paging detail removes the dialog and revoked workspace', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 0.85h', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: 'Worklog details' });
	await expect(dialog.locator('li')).toHaveCount(50);
	state.allowed = [a];
	await dialog.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(dialog).toHaveCount(0);
	await expect(page.getByRole('status').filter({ hasText: 'current access' })).toContainText('current access');
	await expect(page.getByRole('table')).not.toContainText('B-1');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
});

test('all administrative hierarchies preserve totals, scope and current classification', async ({ page }, testInfo) => {
 const state = await fixture(page);
 for (const label of ['By person','By project','By workspace','By team']) {
  await choose(page,'View',label);
  await expect(page.getByRole('table').locator('tfoot')).toContainText('0.51d');
  await expect(scope(page)).not.toContainText('Workspace B (b)');
  await expect(page.getByText('Account unavailable · 00000030',{exact:true})).toBeVisible();
  await expect(page.getByRole('button',{name:'A-1 · 2026-10-20 · 1.50h',exact:true})).toBeVisible();
  await expect(page.getByRole('button',{name:'A-1 · 2026-10-20 · 2.00h',exact:true})).toBeVisible();
  expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe('all');
 }
 await page.screenshot({path:testInfo.outputPath('timesheet-admin-desktop-dark.png'),fullPage:true});
});

test('administrative people filters combine, preserve empty selections and survive mode switches', async ({ page }) => {
 const state = await fixture(page);
 await choose(page,'View','By person');
 await choose(page,'People','Test User · 00000010');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
 await choose(page,'People','Test User · 00000020');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.50d');
 await choose(page,'Projects','No project');
 for (const label of ['By project','By workspace','By team','By person']) {
  await choose(page,'View',label);
  await expect(page.getByRole('table').locator('tfoot')).toContainText('0.06d');
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
 await expect(page.getByRole('tab', { selected: true })).toContainText('By person');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
 expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(otherId);
 expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe(a);
 await choose(page,'People','All');
 await choose(page,'View','By project');
 const cell = page.getByRole('button',{name:'A-1 · 2026-10-20 · 2.00h',exact:true});
 await cell.focus();await page.keyboard.press('Enter');
 const dialog=page.getByRole('dialog');
 await expect(dialog.getByRole('status')).toContainText('0.25d · 1 registrations');
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
 await page.getByRole('button',{name:'A-1 · 2026-10-20 · 2.00h',exact:true}).click();
 await page.getByRole('dialog').getByRole('link').first().click();
 await expect(page).toHaveURL(/\/a\/issue\/A-1$/);
 await page.goBack();
 await expect(page.getByRole('tab', { selected: true })).toContainText('By person');
 await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
 expect(state.gridRequests.at(-1)!.searchParams.get('owner_user_ids')).toBe(otherId);
});

test('administrative pagination retains complete totals and all person options', async ({ page }) => {
 const state=await fixture(page,true,'dark','2026-10-20',0,51);
 await choose(page,'View','By person');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.62d');
 await page.getByRole('button',{name:'Next',exact:true}).last().click();
 await expect(page.getByText('Page 2 of 2')).toBeVisible();
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.62d');
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
 await page.getByRole('button',{name:'A-2 · 2026-10-20 · 0.08h',exact:true}).click();
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
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.51d');
 state.admin=false;
 await page.getByRole('button',{name:'A-1 · 2026-10-20 · 2.00h',exact:true}).click();
 await expect(page.getByRole('dialog')).toHaveCount(0);
 await expect(page.getByRole('tab', { selected: true })).toContainText('My timesheet');
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
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
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.07d');
 const stale=page.waitForResponse(response=>{const url=new URL(response.url());return url.pathname==='/api/timesheets'&&url.searchParams.get('project_ids')==='all';});
 release();await stale;
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.07d');
});

test('an administrative report retains a personal reference outside its data scope', async ({ page }) => {
 const state=await fixture(page);
 await page.goto('/b/timesheet');
 await expect(page.getByRole('button',{name:'Refresh',exact:true})).toBeEnabled();
 await choose(page,'View','By person');
 await expect(page.getByText('Reference: Workspace B (b)',{exact:false})).toBeVisible();
 expect(state.gridRequests.at(-1)!.searchParams.get('reference_workspace_id')).toBe(b);
 expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe(a);
 await expect(page.getByRole('table').locator('tfoot')).toContainText('0.51d');
});

test('a direct administrative URL and reload restore the full view without session state', async ({ page }) => {
	const params = new URLSearchParams({ mode: 'project', period: 'month', from: '2026-10-01', to: '2026-10-31', reference_workspace_id: b, workspace_ids: a, team_ids: 'team-0', project_ids: 'project-0', owner_user_ids: userId, page: '1' });
	const state = await fixture(page, true, 'dark', '2026-10-20', 0, 0, `/a/timesheet?${params}`);
	const assertView = async () => {
		await expect(page.getByRole('tab', { name: 'By project', exact: true })).toHaveAttribute('aria-selected', 'true');
		await expect(page.getByRole('tabpanel', { name: 'By project', exact: true })).toBeVisible();
		await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
		await expect(page.getByRole('table').locator('tfoot')).toContainText('0.19d');
		await expect(scope(page)).not.toContainText('Workspace B');
		for (const [key, value] of params) expect(state.gridRequests.at(-1)!.searchParams.get(key)).toBe(value);
	};
	await assertView();
	const readsBeforeReload = state.gridRequests.length;
	expect(readsBeforeReload).toBeGreaterThan(0);
	const saved = page.url();
	await page.reload();
	await assertView();
	await expect(page).toHaveURL(saved);
	expect(state.gridRequests.length).toBeGreaterThan(readsBeforeReload);
});

test('back and forward restore group page, period and filters without creating new entries', async ({ page }) => {
	const state = await fixture(page, true, 'dark', '2026-10-20', 50);
	const urls = [page.url()];
	await page.getByRole('button', { name: 'Next', exact: true }).click();
	await expect(page.getByText('Page 2 of 2', { exact: true })).toBeVisible();
	urls.push(page.url());
	await page.getByRole('button', { name: 'Month', exact: true }).click();
	await expect(page.getByText('Page 1 of 2', { exact: true })).toBeVisible();
	urls.push(page.url());
	await choose(page, 'Workspaces', 'Workspace B (b)');
	urls.push(page.url());
	await choose(page, 'Projects', 'No project');
	await expect(page.getByText('No work matches the selected filters.')).toBeVisible();
	urls.push(page.url());
	for (let i = urls.length - 2; i >= 0; i--) {
		const before = state.gridRequests.length;
		await page.goBack();
		await expect(page).toHaveURL(urls[i]);
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
		expect(state.gridRequests.length).toBeGreaterThan(before);
		expect(state.gridRequests.at(-1)!.searchParams.get('page')).toBe(i === 1 ? '2' : '1');
		expect(state.gridRequests.at(-1)!.searchParams.get('period')).toBe(i >= 2 ? 'month' : 'week');
	}
	for (const url of urls.slice(1)) {
		await page.goForward();
		await expect(page).toHaveURL(url);
		await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	}
	const count = await page.evaluate(() => history.length);
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	expect(await page.evaluate(() => history.length)).toBe(count);
});

test('mode and people history restores filters and never caches revoked administrative data', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'View', 'By person');
	await choose(page, 'People', 'Test User · 00000020');
	const personUrl = page.url();
	await choose(page, 'View', 'By project');
	const projectUrl = page.url();
	await page.goBack();
	await expect(page).toHaveURL(personUrl);
	await expect(page.getByRole('tab', { name: 'By person', exact: true })).toHaveAttribute('aria-selected', 'true');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
	await page.goForward();
	await expect(page).toHaveURL(projectUrl);
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
	state.admin = false;
	await page.goBack();
	await expect(page.getByRole('tab')).toHaveCount(1);
	await expect(page.getByRole('tab', { selected: true })).toHaveText('My timesheet');
	await expect(page.getByRole('status').filter({ hasText: 'current access' })).toContainText('current access');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.25d');
	await expect(page.getByRole('table')).not.toContainText('Test User');
	expect(new URL(page.url()).searchParams.get('mode')).toBe('personal');
	expect(new URL(page.url()).searchParams.has('owner_user_ids')).toBe(false);
	expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe(a);
});

test('revoked URL selections become empty and a replaced reference preserves historic dates', async ({ page }) => {
	const params = new URLSearchParams({ mode: 'personal', period: 'month', from: '2024-02-01', to: '2024-02-29', reference_workspace_id: b, workspace_ids: b, team_ids: 'team-1', project_ids: 'project-1,none' });
	const state = await fixture(page, true, 'dark', '2026-10-20', 0, 0, `/a/timesheet?${params}`);
	state.allowed = [a];
	await page.reload();
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	await expect(page.getByRole('table')).toHaveCount(0);
	await expect(page.getByRole('status').filter({ hasText: 'current access' })).toContainText('current access');
	await expect(page.getByRole('status').filter({ hasText: 'reference workspace changed' })).toContainText('reference workspace changed');
	const restored = new URL(page.url()).searchParams;
	expect(restored.get('workspace_ids')).toBe('');
	expect(restored.get('team_ids')).toBe('');
	expect(restored.get('project_ids')).toBe('none');
	expect(restored.get('reference_workspace_id')).toBe(a);
	expect(restored.get('from')).toBe('2024-02-01');
	expect(restored.get('to')).toBe('2024-02-29');
});

test('an unauthorized administrative URL cannot expand into all personal workspaces', async ({ page }) => {
	const state = await fixture(page, false, 'dark', '2026-10-20', 0, 0, '/a/timesheet?mode=team&workspace_ids=all');
	await expect(page.getByRole('tab')).toHaveCount(1);
	await expect(page.getByRole('status').filter({ hasText: 'current access' })).toContainText('current access');
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	expect(new URL(page.url()).searchParams.get('workspace_ids')).toBe('');
	expect(state.gridRequests.at(-1)!.searchParams.get('workspace_ids')).toBe('');
});

test('explicit empty people and dimensions remain empty after opening and reloading a URL', async ({ page }) => {
	const state = await fixture(page, true, 'dark', '2026-10-20', 0, 0, '/a/timesheet?mode=person&workspace_ids=&team_ids=&project_ids=none&owner_user_ids=');
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
	const saved = page.url();
	await page.reload();
	await expect(page).toHaveURL(saved);
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const query = state.gridRequests.at(-1)!.searchParams;
	for (const key of ['workspace_ids', 'team_ids', 'owner_user_ids']) expect(query.get(key)).toBe('');
	expect(query.get('project_ids')).toBe('none');
});

test('malformed URLs show an error and reset only on an explicit action', async ({ page }) => {
	const state = await fixture(page);
	const count = state.gridRequests.length;
	for (const query of ['mode=unexpected', 'from=2026-10-20&to=2026-10-26', 'workspace_ids=a,', 'page=0', 'mode=person&mode=project']) {
		await page.goto(`/a/timesheet?${query}`);
		await expect(page.getByText('This Timesheet URL contains invalid parameters.')).toBeVisible();
		await expect(page.getByRole('button', { name: 'Reset view', exact: true })).toBeVisible();
		await expect(page.getByRole('table')).toHaveCount(0);
		expect(state.gridRequests).toHaveLength(count);
	}
	await page.getByRole('button', { name: 'Reset view', exact: true }).click();
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.36d');
	expect(new URL(page.url()).searchParams.get('workspace_ids')).toBe('all');
});

test('manual tabs and the period calendar work with keyboard and retain focus', async ({ page }) => {
	const state = await fixture(page);
	const personal = page.getByRole('tab', { name: 'My timesheet', exact: true });
	const personTab = page.getByRole('tab', { name: 'By person', exact: true });
	await personal.focus();
	await page.keyboard.press('ArrowRight');
	await expect(personTab).toBeFocused();
	await expect(personal).toHaveAttribute('aria-selected', 'true');
	expect([...new Set(state.gridRequests.map(url => url.searchParams.get('mode')))]).toEqual(['personal']);
	await page.keyboard.press('Space');
	await expect(personTab).toHaveAttribute('aria-selected', 'true');
	await expect(page.getByRole('tabpanel', { name: 'By person', exact: true })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await expect(personTab).toBeFocused();
	await page.keyboard.press('End');
	await expect(page.getByRole('tab', { name: 'By team', exact: true })).toBeFocused();
	await page.keyboard.press('Enter');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const trigger = page.getByRole('button', { name: 'Select period', exact: true });
	await trigger.focus();
	await page.keyboard.press('Enter');
	await expect(page.locator('[data-bits-day][data-focused]')).toBeVisible();
	await page.keyboard.press('ArrowUp');
	await page.keyboard.press('Enter');
	await expect(trigger).toBeFocused();
	await expect(page.getByText('No work recorded in this period.')).toBeVisible();
	expect(state.gridRequests.at(-1)!.searchParams.get('from')).toBe('2026-10-12');
});

test('a removed detail cell restores focus to the page heading', async ({ page }) => {
	const state = await fixture(page);
	await choose(page, 'Workspaces', 'Workspace B (b)');
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 0.85h', exact: true }).click();
	await expect(page.getByRole('dialog')).toContainText('51 registrations');
	state.allowed = [a];
	await page.getByRole('dialog').getByRole('button', { name: 'Next', exact: true }).click();
	await expect(page.getByRole('dialog')).toHaveCount(0);
	await expect(page.getByRole('heading', { name: 'Timesheet', exact: true })).toBeFocused();
	await expect(page.getByText('No workspace is selected. Select a workspace to continue.')).toBeVisible();
});

for (const theme of ['light', 'dark'] as const) {
	test(`monthly grid, tabs and detail fit a mobile viewport in ${theme} theme`, async ({ page }, testInfo) => {
		await page.setViewportSize({ width: 390, height: 844 });
		const state = await fixture(page, true, theme);
		await choose(page, 'View', 'By team');
		await page.getByRole('button', { name: 'Month', exact: true }).click();
		const region = page.getByRole('region', { name: 'Timesheet grid' });
		await expect(page.getByRole('table').locator('thead th')).toHaveCount(33);
		expect(await region.evaluate(el => el.scrollWidth > el.clientWidth)).toBe(true);
		await region.focus();
		await page.keyboard.press('ArrowRight');
		await expect.poll(() => region.evaluate(el => el.scrollLeft)).toBeGreaterThan(0);
		expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
		await page.screenshot({ path: testInfo.outputPath(`output/playwright/ep04-timesheet-mobile-${theme}.png`), fullPage: true });
		state.longDetailText = true;
		await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1.50h', exact: true }).click();
		await expect(page.getByRole('dialog')).toContainText('2 registrations');
		const dialog = page.getByRole('dialog');
		expect(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
		await expect(dialog.getByRole('button', { name: 'Close', exact: true }).last()).toBeInViewport();
		const body = dialog.locator('ul').locator('..');
		expect(await body.evaluate(el => el.scrollHeight > el.clientHeight)).toBe(true);
		await page.screenshot({ path: testInfo.outputPath(`output/playwright/ep04-timesheet-detail-${theme}.png`), fullPage: true });
		await body.evaluate(el => el.scrollTop = el.scrollHeight);
		await dialog.getByText('Scope details', { exact: true }).click();
		await expect(dialog.getByText('Reference: Europe/Rome', { exact: true })).toBeInViewport();
		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
		await page.setViewportSize({ width: 1440, height: 1000 });
		await page.screenshot({ path: testInfo.outputPath(`output/playwright/ep04-timesheet-desktop-${theme}.png`), fullPage: true });
	});
}

test('Timesheet errors and detail close controls are translated into Italian', async ({ page }) => {
	const state = await fixture(page, true, 'light', '2026-10-20', 0, 0, '/a/timesheet', 'it');
	await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1,50h', exact: true }).click();
	await expect(page.getByRole('dialog').getByRole('button', { name: 'Chiudi', exact: true })).toHaveCount(2);
	await page.keyboard.press('Escape');
	state.dataFail = true;
	await page.getByRole('button', { name: 'Aggiorna', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Riprova', exact: true })).toBeVisible();
	await page.goto('/a/timesheet?mode=invalid');
	await expect(page.getByText('L’URL del Timesheet contiene parametri non validi.')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Ripristina vista', exact: true })).toBeVisible();
});

test('a page beyond the last group is replaced with a reachable URL', async ({ page }) => {
	const state = await fixture(page, true, 'dark', '2026-10-20', 0, 0, '/a/timesheet?page=99');
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.36d');
	expect(new URL(page.url()).searchParams.get('page')).toBe('1');
	expect(state.gridRequests.map(url => url.searchParams.get('page')).filter((value, index, pages) => index === 0 || value !== pages[index - 1])).toEqual(['99', '1']);
});

test('backend validation errors clear totals and require an explicit reset', async ({ page }) => {
	const state = await fixture(page);
	state.invalidRead = true;
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByText('This Timesheet URL contains invalid parameters.')).toBeVisible();
	await expect(page.getByRole('table')).toHaveCount(0);
	state.invalidRead = false;
	await page.getByRole('button', { name: 'Reset view', exact: true }).click();
	await expect(page.getByRole('table').locator('tfoot')).toContainText('0.36d');
});

test('TSUX searches 100 authorized options without changing scope and explicitly resets empty filters', async ({ page }) => {
	await fixture(page, true, 'light', '2026-10-20', 100);
	const href = page.url();
	await selector(page, 'Workspaces').click();
	await page.getByRole('combobox', { name: 'Search · Workspaces', exact: true }).fill('Extra 99');
	await expect(page.getByRole('option', { name: 'Extra 99 (extra-99)', exact: true })).toBeVisible();
	await expect(page.getByRole('option', { name: 'Workspace A (a)', exact: true })).toHaveCount(0);
	expect(page.url()).toBe(href);
	await page.getByRole('combobox', { name: 'Search · Workspaces', exact: true }).fill('does not exist');
	await expect(page.getByText('No results.', { exact: true })).toBeVisible();
	await page.keyboard.press('Escape');
	expect(page.url()).toBe(href);
	await choose(page, 'Workspaces', 'Workspace A (a)');
	await choose(page, 'Workspaces', 'Workspace A (a)');
	await expect(page.getByText('Workspaces: None selected', { exact: true })).toBeVisible();
	const before = new URL(page.url()).searchParams;
	await page.getByRole('button', { name: 'Reset filters', exact: true }).click();
	await expect(selector(page, 'Workspaces')).toHaveText('All');
	const after = new URL(page.url()).searchParams;
	expect(after.get('from')).toBe(before.get('from'));
	expect(after.get('mode')).toBe(before.get('mode'));
});

for (const width of [320, 390, 768, 1440]) {
	for (const theme of ['light', 'dark'] as const) {
		test(`TSUX grid geometry ${width} ${theme}`, async ({ page }, testInfo) => {
			await page.setViewportSize({ width, height: 1000 });
			await fixture(page, true, theme);
			await page.getByRole('button', { name: 'Month', exact: true }).click();
			await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
			const region = page.getByRole('region', { name: 'Timesheet grid', exact: true });
			await region.scrollIntoViewIfNeeded();
			const viewport = await region.boundingBox();
			const label = await region.locator('thead th').first().boundingBox();
			const day = await region.locator('thead th').nth(1).boundingBox();
			expect(viewport!.width - label!.width).toBeGreaterThanOrEqual(day!.width);
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
			await page.screenshot({ path: testInfo.outputPath(`grid-${width}-${theme}.png`), fullPage: true, animations: 'disabled' });
		});
	}
}

test.describe('TSUX 200 percent browser reflow (1440 physical / 720 CSS pixels)', () => {
	test.use({ viewport: { width: 720, height: 500 }, deviceScaleFactor: 2 });
	for (const locale of ['en', 'it', 'zh-CN', 'zh-TW']) {
		test(`translated report ${locale}`, async ({ page }, testInfo) => {
			await fixture(page, true, 'light', '2026-10-20', 0, 0, '/a/timesheet', locale);
			await expect(page.getByRole('table')).toBeVisible();
			await expect(page.getByRole('table').locator('tfoot td').last()).toHaveText(locale === 'it' ? '0,36d' : '0.36d');
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
			await page.screenshot({ path: testInfo.outputPath(`report-200-${locale}.png`), animations: 'disabled' });
			const cell = page.getByRole('button', { name: `A-1 · 2026-10-20 · ${locale === 'it' ? '1,50h' : '1.50h'}`, exact: true });
			await cell.focus();
			await page.keyboard.press('Enter');
			await expect(page.getByRole('dialog')).toBeVisible();
			expect(await page.getByRole('dialog').evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
			await page.keyboard.press('Escape');
			await expect(cell).toBeFocused();
		});
	}
});
