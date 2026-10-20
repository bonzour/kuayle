import { test, expect, type Page } from '@playwright/test';

const uid = '00000000-0000-0000-0000-000000000010';
const other = '00000000-0000-0000-0000-000000000020';
const wid = (n: number) => `00000000-0000-0000-0000-${String(n).padStart(12, '0')}`;
const person = (id: string) => ({ id, available: true, name: id === uid ? 'Test User' : 'Other Member', display_name: null });
const form = (page: Page) => page.getByRole('dialog', { name: /^(Log work|Edit work log)$/ });
const detail = (page: Page) => page.getByRole('dialog', { name: 'Worklog details', exact: true });

async function fixture(page: Page, enabledA = false) {
	const workspaces = [
		{ id: wid(1), slug: 'a', name: 'Workspace A', timezone: 'Europe/Rome', today: '2026-10-20' },
		{ id: wid(2), slug: 'b', name: 'Workspace B', timezone: 'Pacific/Kiritimati', today: '2026-10-21' }
	];
	const teams = workspaces.map((w, i) => ({ id: wid(100 + i), workspace_id: w.id, name: 'Team', key: i ? 'B' : 'A', icon: null, color: null }));
	const issues = [
		{ id: wid(200), identifier: 'A-1', title: 'Work A1', team: teams[0] },
		...Array.from({ length: 50 }, (_, i) => ({ id: wid(201 + i), identifier: i ? `B-${i + 2}` : 'B-1', title: `Work B${i + 1}`, team: teams[1] })),
		{ id: wid(300), identifier: 'B-2', title: 'Unlogged B2', team: teams[1] }
	];
	const makeLog = (n: number, identifier = 'B-1', owner = uid) => ({ id: wid(1000 + n), issue_id: issues.find(i => i.identifier === identifier)!.id, owner: person(owner), operator: person(uid), work_date: '2026-10-20', duration_minutes: 1, description: `Entry ${n}`, created_at: `2026-10-20T12:00:${String(n % 60).padStart(2, '0')}Z`, updated_at: '2026-10-20T12:00:00Z' });
	const logs = [makeLog(100, 'A-1', other), ...Array.from({ length: 51 }, (_, i) => makeLog(i + 1))];
	const state = {
		enabledA, enabledB: true, roleB: 'member', allowedB: true, todayB: '2026-10-21', logs,
		delegates: [uid, other], revisions: {} as Record<string, number>,
		writes: [] as { path: string; method: string; body: any }[], reads: [] as URL[],
		loseResponse: false, conflict: false, reject: '', failSuggestions: false, revokeOnSettings: false,
		delaySuggestions: null as null | { ready: () => void; wait: Promise<void> },
		receipts: new Map<string, any>()
	};
	const caps = () => workspaces.filter(w => w.slug !== 'b' || state.allowedB).map(w => ({ ...w,
		today: w.slug === 'b' ? state.todayB : w.today,
		efforts_enabled: w.slug === 'a' ? state.enabledA : state.enabledB,
		can_administer: w.slug === 'a' || state.roleB === 'admin',
		can_record_work: w.slug === 'a' ? state.enabledA : state.enabledB && state.roleB !== 'guest'
	}));
	await page.addInitScript(() => localStorage.setItem('PARAGLIDE_LOCALE', 'en'));
	await page.routeWebSocket('**/ws', () => {});
	await page.route('https://raw.githubusercontent.com/carbogninalberto/kuayle/main/UI/static/releases.json', r => r.fulfill({ json: [] }));
	await page.route('**/api/**', async route => {
		const url = new URL(route.request().url());
		const path = url.pathname;
		const error = (code: string, status = 409) => route.fulfill({ status, json: { error: { code } } });
		if (path === '/api/auth/me') return route.fulfill({ json: { ...person(uid), email: 'test@example.test' } });
		if (path === '/api/preferences') return route.fulfill({ json: { theme_mode: 'dark', light_theme: 'light', dark_theme: 'dark', font_size: 'default' } });
		if (path === '/api/workspaces') return route.fulfill({ json: caps().map(w => ({ ...w, current_user_role: w.slug === 'a' ? 'admin' : state.roleB })) });
		if (/^\/api\/workspaces\/[ab]$/.test(path)) return route.fulfill({ json: { ...caps().find(w => w.slug === path.split('/').at(-1)), owner_id: other, current_user_role: 'admin' } });
		if (path.endsWith('/efforts/settings')) {
			const w = caps().find(w => w.slug === path.split('/')[3]);
			if (w?.slug === 'b' && state.revokeOnSettings) { state.allowedB = false; return error('NOT_FOUND', 404); }
			return w ? route.fulfill({ json: { enabled: w.efforts_enabled, timezone: w.timezone, today: w.today, revision: 0 } }) : error('NOT_FOUND', 404);
		}
		const select = (key: string, all: string[]) => !url.searchParams.has(key) || url.searchParams.get(key) === 'all' ? all : url.searchParams.get(key)!.split(',').filter(Boolean);
		if (path === '/api/timesheets/options') {
			const mode = url.searchParams.get('mode') ?? 'personal';
			const ws = caps().filter(w => mode === 'personal' || w.can_administer);
			const selected = select('workspace_ids', ws.map(w => w.id));
			if (selected.some(id => !ws.some(w => w.id === id))) return error('NOT_FOUND', 404);
			return route.fulfill({ json: { mode, available_modes: ['personal', 'person', 'project', 'workspace', 'team'], workspaces: ws, teams: teams.filter(t => selected.includes(t.workspace_id)), projects: [], scope: { workspace_ids: selected, team_ids: teams.filter(t => selected.includes(t.workspace_id)).map(t => t.id), project_ids: [], include_no_project: true } } });
		}
		if (path === '/api/timesheets/issues') {
			state.reads.push(url);
			if (state.failSuggestions) return error('INTERNAL_ERROR', 500);
			const workspace = caps().find(w => w.id === url.searchParams.get('workspace_id'));
			if (!workspace) return error('NOT_FOUND', 404);
			const matches = issues.filter(i => i.team.workspace_id === workspace.id && `${i.identifier} ${i.title}`.toLowerCase().includes((url.searchParams.get('search') ?? '').toLowerCase()));
			const pageNumber = Number(url.searchParams.get('page'));
			if (state.delaySuggestions) { const delay = state.delaySuggestions; state.delaySuggestions = null; delay.ready(); await delay.wait; }
			return route.fulfill({ json: { workspace, data: matches.slice((pageNumber - 1) * 50, pageNumber * 50), total_count: matches.length, page: pageNumber, per_page: 50, has_more: pageNumber * 50 < matches.length } });
		}
		if (path.includes('/issues/') && path.includes('/efforts/')) {
			const slug = path.split('/')[3];
			const identifier = path.split('/')[5];
			const workspace = caps().find(w => w.slug === slug);
			if (!workspace) return error('NOT_FOUND', 404);
			if (path.endsWith('/worklog-owners')) return route.fulfill({ json: state.delegates.map(person) });
			const issue = issues.find(i => i.identifier === identifier && i.team.workspace_id === workspace.id)!;
			const matching = state.logs.filter(l => l.issue_id === issue.id);
			if (route.request().method() === 'GET') {
				state.reads.push(url);
				const p = Number(url.searchParams.get('page') ?? 1);
				return route.fulfill({ json: { data: matching.slice((p - 1) * 50, p * 50), revision: state.revisions[identifier] ?? 0, total_count: matching.length, page: p, per_page: 50, has_more: p * 50 < matching.length } });
			}
			const body = route.request().postDataJSON();
			const method = route.request().method();
			state.writes.push({ path, method, body });
			if (state.reject) return error(state.reject, state.reject === 'FORBIDDEN' ? 403 : 409);
			if (state.roleB === 'guest' && slug === 'b') return error('FORBIDDEN', 403);
			if (state.receipts.has(body.request_id)) return route.fulfill({ json: state.receipts.get(body.request_id) });
			if (!workspace.efforts_enabled) return error('EFFORT_DISABLED');
			if (state.conflict || body.expected_revision !== (state.revisions[identifier] ?? 0)) return error('EFFORT_REVISION_CONFLICT');
			if (method === 'POST') {
				for (const day = new Date(`${body.work_date}T00:00:00Z`); day.toISOString().slice(0, 10) <= body.end_date; day.setUTCDate(day.getUTCDate() + 1)) {
					if (body.skip_weekends && day.getUTCDay() % 6 === 0) continue;
					state.logs.push({ ...makeLog(10000 + state.logs.length, identifier, body.owner_user_id), work_date: day.toISOString().slice(0, 10), duration_minutes: body.duration_minutes, description: body.description });
				}
			} else {
				const index = state.logs.findIndex(l => l.id === path.split('/').at(-1));
				if (method === 'DELETE') state.logs.splice(index, 1);
				else Object.assign(state.logs[index], { work_date: body.work_date, duration_minutes: body.duration_minutes, description: body.description });
			}
			state.revisions[identifier] = (state.revisions[identifier] ?? 0) + 1;
			const receipt = { request_id: body.request_id, revision: state.revisions[identifier], resource_id: issue.id };
			state.receipts.set(body.request_id, receipt);
			if (state.loseResponse) { state.loseResponse = false; return route.abort('failed'); }
			return route.fulfill({ json: receipt });
		}
		if (path === '/api/timesheets' || path === '/api/timesheets/worklogs') {
			const mode = url.searchParams.get('mode');
			const ws = caps().filter(w => (mode === 'personal' || w.can_administer) && select('workspace_ids', caps().map(w => w.id)).includes(w.id));
			const from = url.searchParams.get('from')!;
			const to = url.searchParams.get('to')!;
			const days: string[] = [];
			for (let day = new Date(`${from}T00:00:00Z`); day.toISOString().slice(0, 10) <= to; day.setUTCDate(day.getUTCDate() + 1)) days.push(day.toISOString().slice(0, 10));
			const detailRead = path.endsWith('/worklogs');
			const records = state.logs.filter(l => {
				const issue = issues.find(i => i.id === l.issue_id)!;
				return ws.some(w => w.id === issue.team.workspace_id) && l.work_date >= from && l.work_date <= to && (mode !== 'personal' || l.owner.id === uid) && select('owner_user_ids', [uid, other]).includes(l.owner.id)
					&& (!detailRead || l.issue_id === url.searchParams.get('issue_id') && l.work_date === url.searchParams.get('work_date'));
			});
			const daily = (rows: typeof logs) => days.map(date => ({ date, minutes: rows.filter(l => l.work_date === date).reduce((sum, l) => sum + l.duration_minutes, 0) }));
			const total = records.reduce((sum, l) => sum + l.duration_minutes, 0);
			const p = Number(url.searchParams.get('page'));
			const common = { mode, period: url.searchParams.get('period'), from, to, days, reference: caps()[0], workspaces: ws, scope: { workspace_ids: ws.map(w => w.id), team_ids: [], project_ids: [], include_no_project: true }, daily_totals: daily(records), total_minutes: total, page: p, per_page: 50, has_more: false };
			if (detailRead) return route.fulfill({ json: { ...common, total_count: records.length, has_more: p * 50 < records.length, data: records.slice((p - 1) * 50, p * 50).map(l => { const issue = issues.find(i => i.id === l.issue_id)!; return { ...l, issue_identifier: issue.identifier, issue_title: issue.title, team: issue.team, workspace: ws.find(w => w.id === issue.team.workspace_id) }; }) } });
			const ownerGroups = (mode === 'personal' ? [uid] : [...new Set(records.map(l => l.owner.id))]).map(ownerId => {
			const ownerRecords = records.filter(l => l.owner.id === ownerId);
			const groups = ws.map(w => ({ key: w.id, id: w.id, name: w.name, type: 'workspace', workspace_id: w.id, workspace_slug: w.slug, daily_totals: daily(ownerRecords.filter(l => issues.find(i => i.id === l.issue_id)!.team.workspace_id === w.id)), total_minutes: total, children: issues.filter(i => i.team.workspace_id === w.id && ownerRecords.some(l => l.issue_id === i.id)).map(i => ({ key: i.id, id: i.id, name: i.title, type: 'issue', identifier: i.identifier, workspace_id: w.id, workspace_slug: w.slug, daily_totals: daily(ownerRecords.filter(l => l.issue_id === i.id)), total_minutes: ownerRecords.filter(l => l.issue_id === i.id).reduce((sum, l) => sum + l.duration_minutes, 0), children: [] })) })).filter(g => g.children.length);
			return mode === 'personal' ? groups : [{ key: ownerId, id: ownerId, name: person(ownerId).name, person: person(ownerId), type: 'person', workspace_id: null, workspace_slug: null, daily_totals: daily(ownerRecords), total_minutes: ownerRecords.reduce((sum, l) => sum + l.duration_minutes, 0), children: groups }];
			});
			const groups = ownerGroups.flat();
			return route.fulfill({ json: { ...common, groups, people: [person(uid), person(other)], total_count: groups.length, empty_reason: null } });
		}
		return route.fulfill({ json: [] });
	});
	await page.goto('/a/timesheet');
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	return state;
}

async function openGlobal(page: Page) {
	await page.getByRole('button', { name: 'Log work', exact: true }).click();
	await form(page).getByLabel('Workspaces', { exact: true }).selectOption(wid(2));
	await expect(form(page).getByRole('button', { name: 'B-1 · Work B1 Team', exact: true })).toBeVisible();
}

async function fill(page: Page, duration = '1h', description = 'Timesheet draft') {
	await form(page).getByLabel('Duration', { exact: true }).fill(duration);
	await form(page).getByLabel('Description (optional)', { exact: true }).fill(description);
}

test('global entry finds an unlogged issue beyond page one and retains administrative filters', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('tab', { name: 'By person', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	const href = page.url();
	await openGlobal(page);
	await form(page).getByRole('button', { name: 'Next', exact: true }).click();
	await form(page).getByRole('button', { name: 'B-2 · Unlogged B2 Team', exact: true }).click();
	await expect(form(page).getByLabel('From', { exact: true })).toHaveValue('2026-10-21');
	await expect(form(page)).toContainText('Workspace B (b) · B-2 · Pacific/Kiritimati');
	await fill(page, '1d');
	await form(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(form(page)).toHaveCount(0);
	await expect(page.getByRole('status')).toContainText('Work saved');
	expect(page.url()).toBe(href);
	expect(state.writes[0].path).toBe('/api/workspaces/b/issues/B-2/efforts/worklogs');
	expect(state.writes[0].body.duration_minutes).toBe(480);
});

test('empty cells prefill the destination and date; retry preserves the period without duplicates', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'Log work · B-1 · 2026-10-19', exact: true }).click();
	await expect(form(page).getByLabel('From', { exact: true })).toHaveValue('2026-10-19');
	await form(page).getByLabel('To', { exact: true }).fill('2026-10-20');
	await fill(page, '1h 30m');
	state.loseResponse = true;
	await form(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(form(page).getByRole('button', { name: 'Retry save', exact: true })).toBeEnabled();
	await expect(form(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('Timesheet draft');
	await form(page).getByRole('button', { name: 'Retry save', exact: true }).click();
	await expect(form(page)).toHaveCount(0);
	expect(state.writes[0].body).toEqual(state.writes[1].body);
	expect(state.logs.filter(l => l.description === 'Timesheet draft')).toHaveLength(2);
	await expect(page.getByRole('button', { name: 'B-1 · 2026-10-19 · 1h 30m', exact: true })).toBeVisible();
});

test('detail edits reread beyond 50 records, update both days, and delete with confirmation', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true }).click();
	await detail(page).getByRole('button', { name: 'Next', exact: true }).click();
	await detail(page).getByRole('button', { name: 'Edit work log', exact: true }).click();
	await expect(form(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('Entry 51');
	expect(state.reads.some(u => u.pathname.endsWith('/efforts/worklogs') && u.searchParams.get('page') === '2')).toBe(true);
	await form(page).getByLabel('Work date', { exact: true }).fill('2026-10-19');
	await fill(page, '2h', 'Corrected');
	await form(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(form(page)).toHaveCount(0);
	await expect(detail(page).getByText('Entry 1', { exact: true })).toBeVisible();
	await detail(page).getByRole('button', { name: 'Close', exact: true }).last().click();
	await page.getByRole('button', { name: 'B-1 · 2026-10-19 · 2h', exact: true }).click();
	await detail(page).getByRole('button', { name: 'Delete', exact: true }).click();
	const confirmation = page.getByRole('dialog', { name: 'Delete work log?', exact: true });
	await expect(confirmation).toContainText('Corrected');
	await confirmation.getByRole('button', { name: 'Delete', exact: true }).click();
	await expect(confirmation).toHaveCount(0);
	await expect(detail(page)).toContainText('No work matches the selected filters.');
	expect(state.logs.some(l => l.description === 'Corrected')).toBe(false);
});

test('detail can add work and conflict handling preserves the draft until explicit realignment', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true }).click();
	await detail(page).getByRole('button', { name: 'Log work', exact: true }).click();
	await fill(page);
	state.conflict = true;
	await form(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(form(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('Timesheet draft');
	await expect(form(page).getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
	state.conflict = false;
	await form(page).getByRole('button', { name: 'Replace draft with saved values', exact: true }).click();
	await expect(form(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('');
});

for (const rejection of ['disabled', 'guest', 'revoked'] as const) {
	test(`a ${rejection} destination rejects an open form and keeps its draft`, async ({ page }) => {
		const state = await fixture(page);
		await page.getByRole('button', { name: 'Log work · B-1 · 2026-10-19', exact: true }).click();
		await fill(page);
		if (rejection === 'disabled') state.enabledB = false;
		if (rejection === 'guest') state.roleB = 'guest';
		if (rejection === 'revoked') state.allowedB = false;
		await form(page).getByRole('button', { name: 'Save', exact: true }).click();
		await expect(form(page).getByRole('alert')).toBeVisible();
		await expect(form(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('Timesheet draft');
		await expect(form(page).getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
	});
}

test('delegation uses destination candidates and explains fallback for a removed person', async ({ page }) => {
	const state = await fixture(page, true);
	await page.getByRole('tab', { name: 'By person', exact: true }).click();
	await page.getByRole('button', { name: 'A-1 · 2026-10-20 · 1m', exact: true }).click();
	await detail(page).getByRole('button', { name: 'Log work', exact: true }).click();
	await expect(form(page).getByLabel('Work owner', { exact: true })).toHaveValue(other);
	await form(page).getByRole('button', { name: 'Cancel', exact: true }).click();
	state.delegates = [uid];
	await detail(page).getByRole('button', { name: 'Log work', exact: true }).click();
	await expect(form(page)).toContainText('The owner is set to you');
	await expect(form(page).getByLabel('Work owner', { exact: true })).toHaveValue(uid);
});

test('search has retry and discards late responses after workspace changes', async ({ page }) => {
	const state = await fixture(page, true);
	await openGlobal(page);
	state.failSuggestions = true;
	await form(page).getByRole('searchbox', { name: 'Search issues', exact: true }).fill('Unlogged');
	await expect(form(page).getByRole('button', { name: 'Try again', exact: true })).toBeVisible();
	state.failSuggestions = false;
	await form(page).getByRole('button', { name: 'Try again', exact: true }).click();
	await expect(form(page).getByRole('button', { name: 'B-2 · Unlogged B2 Team', exact: true })).toBeVisible();
	let release!: () => void;
	let ready!: () => void;
	const entered = new Promise<void>(resolve => ready = resolve);
	state.delaySuggestions = { ready, wait: new Promise<void>(resolve => release = resolve) };
	await form(page).getByRole('searchbox', { name: 'Search issues', exact: true }).fill('B-1');
	await entered;
	await form(page).getByLabel('Workspaces', { exact: true }).selectOption(wid(1));
	await expect(form(page).getByRole('button', { name: 'A-1 · Work A1 Team', exact: true })).toBeVisible();
	const late = page.waitForResponse(response => {
		const url = new URL(response.url());
		return url.pathname === '/api/timesheets/issues' && url.searchParams.get('search') === 'B-1';
	});
	release();
	await (await late).finished();
	await page.evaluate(() => new Promise<void>(resolve => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
	await expect(form(page).getByRole('button', { name: 'A-1 · Work A1 Team', exact: true })).toBeVisible();
	await expect(form(page).getByRole('button', { name: 'B-1 · Work B1 Team', exact: true })).toHaveCount(0);
});

test('destination today validates future dates and readonly capabilities remain local', async ({ page }) => {
	const state = await fixture(page);
	await page.getByRole('button', { name: 'Log work · B-1 · 2026-10-19', exact: true }).click();
	await expect(form(page).getByLabel('From', { exact: true })).toHaveAttribute('max', '2026-10-21');
	await form(page).getByLabel('From', { exact: true }).fill('2026-10-22');
	await form(page).getByLabel('To', { exact: true }).fill('2026-10-22');
	await fill(page);
	await form(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(form(page).getByLabel('From', { exact: true })).toHaveAttribute('aria-invalid', 'true');
	expect(state.writes).toHaveLength(0);
	await form(page).getByRole('button', { name: 'Cancel', exact: true }).click();
	state.roleB = 'guest';
	await page.getByRole('button', { name: 'Refresh', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true })).toBeEnabled();
	await expect(page.getByRole('button', { name: 'Log work', exact: true })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'Log work · B-1 · 2026-10-19', exact: true })).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'B-1 · 2026-10-20 · 51m', exact: true })).toBeVisible();
});

test('mobile entry supports keyboard and restores focus after cancellation', async ({ page }, testInfo) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await fixture(page);
	const origin = page.getByRole('button', { name: 'Log work · B-1 · 2026-10-19', exact: true });
	await origin.focus();
	await page.keyboard.press('Enter');
	await expect(form(page).getByLabel('From', { exact: true })).toHaveValue('2026-10-19');
	await fill(page);
	await form(page).screenshot({ path: testInfo.outputPath('ep05-mobile-worklog.png') });
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
	await page.keyboard.press('Escape');
	await expect(form(page)).toHaveCount(0);
	await expect(origin).toBeFocused();
});

test('revocation discovered while opening a form clears the old grid without closing the form', async ({ page }) => {
	const state = await fixture(page);
	state.revokeOnSettings = true;
	await page.getByRole('button', { name: 'Log work · B-1 · 2026-10-19', exact: true }).click();
	await expect(form(page).getByRole('alert')).toBeVisible();
	await expect(form(page).getByLabel('From', { exact: true })).toHaveValue('2026-10-19');
	await expect(form(page).getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
	await expect(page.getByRole('button', { name: 'Refresh', exact: true, includeHidden: true })).toBeEnabled();
	await expect(page.locator('button[aria-label="Log work · B-1 · 2026-10-19"]')).toHaveCount(0);
	expect(state.writes).toHaveLength(0);
});
