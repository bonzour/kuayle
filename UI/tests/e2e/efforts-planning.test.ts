import { test, expect, type Page, type WebSocketRoute } from '@playwright/test';

const userId = '00000000-0000-0000-0000-000000000001';
const teamId = '00000000-0000-0000-0000-000000000010';
const projectId = '00000000-0000-0000-0000-000000000030';
const statusId = '00000000-0000-0000-0000-000000000020';
const user = { id: userId, name: 'Test User', email: 'test@example.com', avatar_url: null };
const timestamp = '2026-10-19T12:00:00Z';
const planning = (page: Page) => page.getByRole('region', { name: 'Effort', exact: true });
type Field = 'Estimate' | 'Remaining';
const field = (page: Page, name: Field) => planning(page).getByRole('group', { name, exact: true });
const value = (page: Page, name: Field) => field(page, name).getByRole('button', { name, exact: true });
const input = (page: Page, name: Field) => field(page, name).getByRole('textbox');
async function edit(page: Page, name: Field, text: string) {
	if (!await input(page, name).isVisible()) await value(page, name).click();
	await input(page, name).fill(text);
}
async function enter(page: Page, name: Field, text: string) {
	await edit(page, name, text);
	await input(page, name).press('Enter');
}
const blank = () => ({ revision: 0, own: { estimate_minutes: null as number | null, spent_minutes: 120, remaining_minutes: null as number | null }, total: { estimate_minutes: null, spent_minutes: 120, remaining_minutes: 0, estimate_partial: true, remaining_partial: true } });

function fixture(role = 'owner') {
	const issues = [1, 2].map(number => ({
		id: `00000000-0000-0000-0000-00000000010${number}`, identifier: `EP-${number}`, title: `Planning issue ${number}`, description: null,
		status: 'todo', status_id: statusId, status_info: { id: statusId, name: 'Todo', category: 'unstarted', color: null, position: 0 },
		priority: 0, team_id: teamId, project_id: projectId, cycle_id: null, creator_id: userId,
		assignee_id: null as string | null, assignees: [user], parent_id: null, due_date: null, sort_order: number, labels: [],
		created_at: timestamp, updated_at: timestamp
	}));
	const project = { id: projectId, name: 'Planning project', team_id: teamId, status: 'planned', start_date: null, target_date: null, description: null, created_at: timestamp, updated_at: timestamp };
	const team = { id: teamId, name: 'Engineering', key: 'EP', color: '#6366f1', icon: 'layers', created_at: timestamp, updated_at: timestamp };
	const summaries = { 'EP-1': blank(), 'EP-2': { ...blank(), own: { estimate_minutes: 180, spent_minutes: 0, remaining_minutes: 180 } } };
	const state = {
		role, issues, summaries, enabled: true, failRead: false, loseResponse: false, conflict: false, writeStatus: 0,
		delayMembership: null as null | { ready: () => void; wait: Promise<void> },
		delayRead: null as null | { ready: () => void; wait: Promise<void> },
		writes: [] as { path: string; body: Record<string, any> }[], receipts: new Map<string, Record<string, unknown>>(), sockets: [] as WebSocketRoute[],
		refresh() { for (const socket of state.sockets) socket.send(JSON.stringify({ type: 'app.refresh', payload: { resources: ['workspace', 'issues'], privacy: true } })); }
	};
	async function attach(page: Page, view: 'page' | 'panel' = 'page') {
		await page.addInitScript(() => localStorage.setItem('PARAGLIDE_LOCALE', 'en'));
		await page.routeWebSocket('**/ws', socket => { state.sockets.push(socket); });
		await page.route('https://raw.githubusercontent.com/carbogninalberto/kuayle/main/UI/static/releases.json', route => route.fulfill({ json: [] }));
		await page.route(url => url.pathname.startsWith('/api/'), async route => {
			const path = new URL(route.request().url()).pathname;
			if (path === '/api/auth/me') return route.fulfill({ json: user });
			if (path === '/api/preferences') return route.fulfill({ json: {} });
			const workspace = { id: 'test-id', slug: 'test', name: 'Test workspace', current_user_role: state.role, owner_id: userId, created_at: timestamp, updated_at: timestamp };
			if (path === '/api/workspaces') return route.fulfill({ json: [workspace] });
			if (path === '/api/workspaces/test') {
				const delay = state.delayMembership;
				if (delay) { state.delayMembership = null; delay.ready(); await delay.wait; }
				return route.fulfill({ json: workspace });
			}
			if (path.endsWith('/efforts/settings')) return route.fulfill({ json: { enabled: state.enabled, timezone: 'Europe/Rome', revision: 0, today: '2026-10-19' } });
			const match = path.match(/\/issues\/(EP-[12])\/efforts(?:\/(estimate|remaining))?$/);
			if (match) {
				const summary = summaries[match[1] as keyof typeof summaries];
				if (route.request().method() === 'GET') {
					if (state.failRead) return route.fulfill({ status: 500, json: { error: { code: 'INTERNAL_ERROR' } } });
					const value = structuredClone(summary);
					const delay = state.delayRead;
					if (delay) { state.delayRead = null; delay.ready(); await delay.wait; }
					return route.fulfill({ json: value });
				}
				const body = route.request().postDataJSON();
				state.writes.push({ path, body });
				if (state.writeStatus) return route.fulfill({ status: state.writeStatus, json: { error: { code: state.writeStatus === 403 ? 'FORBIDDEN' : 'NOT_FOUND' } } });
				if (state.conflict) {
					state.conflict = false;
					summary.revision++;
					summary.own.estimate_minutes = 60;
					return route.fulfill({ status: 409, json: { error: { code: 'EFFORT_REVISION_CONFLICT' } } });
				}
				let receipt = state.receipts.get(body.request_id);
				if (!receipt) {
					if (body.expected_revision !== summary.revision) return route.fulfill({ status: 409, json: { error: { code: 'EFFORT_REVISION_CONFLICT' } } });
					if (match[2] === 'estimate') {
						summary.own.estimate_minutes = body.estimate_minutes;
						if (summary.own.remaining_minutes === null && body.estimate_minutes !== null) summary.own.remaining_minutes = Math.max(body.estimate_minutes - summary.own.spent_minutes, 0);
					} else summary.own.remaining_minutes = body.remaining_minutes;
					receipt = { request_id: body.request_id, revision: ++summary.revision, resource_id: issues[0].id };
					state.receipts.set(body.request_id, receipt);
				}
				if (state.loseResponse) { state.loseResponse = false; return route.abort('failed'); }
				return route.fulfill({ json: receipt });
			}
			const issueMatch = path.match(/\/issues\/(EP-[12])$/);
			if (issueMatch) return route.fulfill({ json: issues.find(issue => issue.identifier === issueMatch[1]) });
			if (path.endsWith('/issues')) return route.fulfill({ json: { data: issues, total_count: 2, page: 1, per_page: 50, has_more: false } });
			if (path.endsWith(`/projects/${projectId}`)) return route.fulfill({ json: project });
			if (path.endsWith('/projects')) return route.fulfill({ json: [project] });
			if (path.endsWith('/teams')) return route.fulfill({ json: [team] });
			if (path.endsWith('/statuses')) return route.fulfill({ json: [{ ...issues[0].status_info, team_id: teamId, slug: 'todo', is_default: true }] });
			if (path.endsWith('/github') || path.endsWith('/github/status')) return route.fulfill({ json: { connected: false, repos: [], pull_requests: [], branches: [], commits: [] } });
			if (path.endsWith('/dev-machine-scope-setting')) return route.fulfill({ json: { github_repo_id: null, environment_id: null } });
			if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
			if (['/comments', '/history', '/relations', '/sub-issues', '/members', '/labels', '/views', '/cycles', '/dev-machine-environments'].some(suffix => path.endsWith(suffix))) return route.fulfill({ json: [] });
			return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND', message: path } } });
		});
		await page.goto(view === 'panel' ? `/test/projects/${projectId}` : '/test/issue/EP-1');
		if (view === 'panel') await page.getByText('Planning issue 1', { exact: true }).click();
		await expect(value(page, 'Estimate')).toHaveText('Not estimated');
		await expect(value(page, 'Remaining')).toHaveText('0m (Not set)');
	}
	return { state, attach };
}

for (const view of ['page', 'panel'] as const) {
	test(`compact Effort rows autosave in the ${view}`, async ({ page }, testInfo) => {
		const { state, attach } = fixture();
		await attach(page, view);
		await expect(planning(page).getByRole('button', { name: 'Effort', exact: true })).toHaveAttribute('aria-expanded', 'true');
		await expect(planning(page).getByRole('textbox')).toHaveCount(0);
		await expect(planning(page).getByRole('button', { name: 'Save', exact: true })).toHaveCount(0);
		await expect(planning(page).getByLabel('Time spent', { exact: true })).toHaveText('2h');
		await expect(field(page, 'Estimate').locator('svg')).toHaveCount(0);
		await expect(field(page, 'Remaining').locator('svg')).toHaveCount(0);
		if (view === 'page') {
			expect(await planning(page).evaluate(element => element.previousElementSibling?.querySelector('button')?.textContent?.trim())).toBe('Project');
		}
		await edit(page, 'Estimate', '8h');
		await expect(input(page, 'Estimate')).toBeFocused();
		await expect(page.locator('[data-slot="popover-content"][data-state="open"]')).toHaveCount(0);
		await page.screenshot({ path: testInfo.outputPath('effort-inline.png') });
		await testInfo.attach('Inline effort editor', { path: testInfo.outputPath('effort-inline.png'), contentType: 'image/png' });
		await input(page, 'Estimate').press('Enter');
		await expect(value(page, 'Estimate')).toHaveText('8h');
		await expect(value(page, 'Remaining')).toHaveText('6h');
		await expect(page.locator('[data-slot="popover-content"][data-state="open"]')).toHaveCount(0);
		await expect(input(page, 'Estimate')).toHaveCount(0);
		await planning(page).scrollIntoViewIfNeeded();
		await page.screenshot({ path: testInfo.outputPath('effort.png') });
		await testInfo.attach('Effort section', { path: testInfo.outputPath('effort.png'), contentType: 'image/png' });
		await edit(page, 'Estimate', '2h');
		await page.getByText('Status', { exact: true }).last().click();
		await expect(value(page, 'Estimate')).toHaveText('2h');
		await expect(value(page, 'Remaining')).toHaveText('6h');
		await value(page, 'Estimate').click();
		await field(page, 'Estimate').getByRole('button', { name: 'Remove estimate' }).click();
		await expect(value(page, 'Estimate')).toHaveText('Not estimated');
		await enter(page, 'Remaining', '1h 30m');
		await expect(value(page, 'Remaining')).toHaveText('1h 30m');
		expect(state.summaries['EP-1'].own).toEqual({ estimate_minutes: null, spent_minutes: 120, remaining_minutes: 90 });
		await enter(page, 'Estimate', '0');
		await expect(value(page, 'Estimate')).toHaveText('0m');
		expect(state.summaries['EP-1'].own.remaining_minutes).toBe(90);
		expect(state.writes.map(write => write.body.expected_revision)).toEqual([0, 1, 2, 3, 4]);
	});
}

for (const role of ['owner', 'admin', 'member', 'guest']) {
	test(`planning controls respect ${role} permissions`, async ({ page }) => {
		const { state, attach } = fixture(role);
		if (role === 'owner' || role === 'admin') state.issues[0].assignees = [];
		await attach(page);
		if (role === 'guest') await expect(value(page, 'Estimate')).toBeDisabled();
		else {
			await expect(value(page, 'Estimate')).toBeEnabled();
			await enter(page, 'Remaining', '0m');
			await expect.poll(() => state.summaries['EP-1'].own.remaining_minutes).toBe(0);
		}
	});
}

test('invalid input, conflict and realignment preserve drafts', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await enter(page, 'Estimate', '1.5h');
	await expect(input(page, 'Estimate')).toHaveAttribute('aria-invalid', 'true');
	expect(state.writes).toHaveLength(0);
	state.conflict = true;
	await enter(page, 'Estimate', '8h');
	await expect(field(page, 'Estimate').getByRole('alert')).toContainText('Effort has changed');
	await expect(input(page, 'Estimate')).toHaveValue('8h');
	await field(page, 'Estimate').getByRole('button', { name: 'Replace draft with saved values' }).click();
	await expect(value(page, 'Estimate')).toHaveText('1h');
	await value(page, 'Estimate').click();
	await expect(input(page, 'Estimate')).toHaveValue('1h');
});

test('lost response retries the exact request after a remote refresh', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	state.loseResponse = true;
	await enter(page, 'Estimate', '8h');
	await expect(field(page, 'Estimate').getByRole('alert')).toContainText('could not be confirmed');
	state.refresh();
	await field(page, 'Estimate').getByRole('button', { name: 'Retry save' }).click();
	await expect(field(page, 'Estimate').getByRole('alert')).toHaveCount(0);
	expect(state.writes).toHaveLength(2);
	expect(state.writes[0]).toEqual(state.writes[1]);
	expect(state.summaries['EP-1'].revision).toBe(1);
});

test('remote disabling, closing and revoking assignment update controls', async ({ page }) => {
	const { state, attach } = fixture('member');
	await attach(page);
	await expect(value(page, 'Estimate')).toBeEnabled();
	state.enabled = false;
	state.refresh();
	await expect(planning(page)).toContainText('Efforts is disabled');
	await expect(value(page, 'Estimate')).toBeDisabled();
	state.enabled = true;
	state.issues[0].status_info.category = 'completed';
	state.refresh();
	await expect(value(page, 'Estimate')).toBeEnabled();
	await expect(value(page, 'Remaining')).toBeDisabled();
	await expect(planning(page)).toContainText('cannot be edited on completed');
	state.issues[0].assignees = [];
	state.refresh();
	await expect(value(page, 'Estimate')).toBeDisabled();
});

test('failed reload preserves a draft and slower reads cannot overwrite newer values', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await edit(page, 'Estimate', '8h');
	state.failRead = true;
	state.refresh();
	await expect(planning(page).getByRole('alert')).toContainText('Unable to load');
	await expect(input(page, 'Estimate')).toHaveValue('8h');
	await page.getByText('Status', { exact: true }).last().click();
	state.failRead = false;
	await planning(page).getByRole('button', { name: 'Reload', exact: true }).click();
	await expect(planning(page).getByRole('alert')).toHaveCount(0);
	await expect(input(page, 'Estimate')).toHaveValue('8h');
	let release!: () => void;
	const waiting = new Promise<void>(resolve => { release = resolve; });
	let ready!: () => void;
	const started = new Promise<void>(resolve => { ready = resolve; });
	state.delayRead = { ready, wait: waiting };
	state.refresh();
	await started;
	state.summaries['EP-1'].revision = 3;
	state.summaries['EP-1'].own.remaining_minutes = 180;
	state.refresh();
	await expect(value(page, 'Remaining')).toHaveText('3h');
	const lateResponse = page.waitForResponse(async response => response.url().endsWith('/EP-1/efforts') && (await response.json()).revision === 0);
	release();
	await lateResponse;
	await expect(value(page, 'Remaining')).toHaveText('3h');
	await expect(input(page, 'Estimate')).toHaveValue('8h');
	await expect(field(page, 'Estimate').getByRole('status')).toContainText('Effort has changed');
});

test('late responses from a previous issue do not replace the new issue', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	let release!: () => void;
	const wait = new Promise<void>(resolve => { release = resolve; });
	let ready!: () => void;
	const started = new Promise<void>(resolve => { ready = resolve; });
	state.delayRead = { ready, wait };
	state.refresh();
	await started;
	await page.getByRole('button', { name: 'Next issue (J)' }).click();
	await expect(page).toHaveURL('/test/issue/EP-2');
	await expect(value(page, 'Estimate')).toHaveText('3h');
	const lateResponse = page.waitForResponse(response => response.url().endsWith('/EP-1/efforts'));
	release();
	await lateResponse;
	await expect(value(page, 'Estimate')).toHaveText('3h');
	await expect(value(page, 'Remaining')).toHaveText('3h');
});

test('Escape cancels edits and the Effort card collapses like Details', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await edit(page, 'Estimate', '8h');
	await input(page, 'Estimate').press('Escape');
	await expect(input(page, 'Estimate')).toHaveCount(0);
	await expect(value(page, 'Estimate')).toHaveText('Not estimated');
	await expect(value(page, 'Estimate')).toBeFocused();
	expect(state.writes).toHaveLength(0);
	await value(page, 'Estimate').click();
	await expect(input(page, 'Estimate')).toHaveValue('');
	await input(page, 'Estimate').press('Escape');
	const heading = planning(page).getByRole('button', { name: 'Effort', exact: true });
	await heading.click();
	await expect(heading).toHaveAttribute('aria-expanded', 'false');
	await expect(value(page, 'Estimate')).not.toBeVisible();
	await heading.click();
	await expect(value(page, 'Estimate')).toBeVisible();
	await enter(page, 'Estimate', '1h');
	await expect(value(page, 'Estimate')).toHaveText('1h');
	await enter(page, 'Estimate', '1.5h');
	await expect(input(page, 'Estimate')).toHaveAttribute('aria-invalid', 'true');
	await enter(page, 'Estimate', '1h');
	await expect(input(page, 'Estimate')).toHaveCount(0);
	await expect(field(page, 'Estimate').getByRole('alert')).toHaveCount(0);
	expect(state.writes).toHaveLength(1);
});

for (const status of [403, 404]) {
	test(`write ${status} reports the correct error and preserves the inline draft`, async ({ page }) => {
		const { state, attach } = fixture();
		await attach(page);
		state.writeStatus = status;
		await enter(page, 'Estimate', '8h');
		await expect(field(page, 'Estimate').getByRole('alert')).toContainText(status === 403 ? 'Only workspace owners' : 'currently unavailable');
		await expect(input(page, 'Estimate')).toHaveValue('8h');
		if (status === 404) await expect(planning(page)).not.toContainText('Only workspace owners');
		expect(state.summaries['EP-1'].revision).toBe(0);
	});
}

test('an owner awaiting membership refresh is not reported as unauthorized', async ({ page }) => {
	const { state, attach } = fixture();
	state.issues[0].assignees = [];
	await attach(page);
	let release!: () => void;
	const wait = new Promise<void>(resolve => { release = resolve; });
	let ready!: () => void;
	const started = new Promise<void>(resolve => { ready = resolve; });
	state.delayMembership = { ready, wait };
	state.refresh();
	await started;
	await expect(value(page, 'Estimate')).toBeDisabled();
	await expect(planning(page)).not.toContainText('Only workspace owners');
	release();
	await expect(value(page, 'Estimate')).toBeEnabled();
	await enter(page, 'Estimate', '1h');
	await expect(value(page, 'Estimate')).toHaveText('1h');
});
