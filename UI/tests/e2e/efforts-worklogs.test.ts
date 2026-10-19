import { test, expect, type Page, type WebSocketRoute } from '@playwright/test';

const userId = '00000000-0000-0000-0000-000000000001';
const memberId = '00000000-0000-0000-0000-000000000002';
const teamId = '00000000-0000-0000-0000-000000000010';
const projectId = '00000000-0000-0000-0000-000000000030';
const statusId = '00000000-0000-0000-0000-000000000020';
const user = { id: userId, name: 'Test User', display_name: 'Test User', email: 'test@example.com', avatar_url: null };
const person = (id = userId) => ({ id, available: true, name: id === userId ? 'Test User' : 'Other Member', display_name: null });
const timestamp = '2026-10-19T12:00:00Z';
const register = (page: Page) => page.getByRole('region', { name: 'Work log', exact: true });
const dialog = (page: Page) => page.locator('[data-slot="dialog-content"]');
const rows = (page: Page) => register(page).getByRole('listitem');

function fixture(role = 'member', count = 0) {
	const issues = [1, 2].map((number) => ({
		id: `00000000-0000-0000-0000-00000000010${number}`,
		identifier: `EP-${number}`,
		title: `Worklog issue ${number}`,
		description: null,
		status: 'todo',
		status_id: statusId,
		status_info: { id: statusId, name: 'Todo', category: 'unstarted', color: null, position: 0 },
		priority: 0,
		team_id: teamId,
		project_id: projectId,
		cycle_id: null,
		creator_id: userId,
		assignee_id: null,
		assignees: [],
		parent_id: null,
		due_date: null,
		sort_order: number,
		labels: [],
		created_at: timestamp,
		updated_at: timestamp
	}));
	const makeLog = (i: number, identifier = 'EP-1') => ({
		id: `10000000-0000-0000-0000-${String(i).padStart(12, '0')}`,
		issue_id: issues[identifier === 'EP-1' ? 0 : 1].id,
		owner: person(),
		operator: person(),
		work_date: '2026-01-01',
		duration_minutes: 1,
		description: `Entry ${i}`,
		created_at: timestamp,
		updated_at: timestamp
	});
	const logs: Record<string, ReturnType<typeof makeLog>[]> = { 'EP-1': Array.from({ length: count }, (_, i) => makeLog(i + 1)), 'EP-2': [] };
	const revisions: Record<string, number> = { 'EP-1': 0, 'EP-2': 0 };
	const remaining: Record<string, number> = { 'EP-1': 480, 'EP-2': 480 };
	const state = {
		role,
		enabled: true,
		today: '2026-01-02',
		loseResponse: false,
		conflict: false,
		dailyLimit: false,
		writeStatus: 0,
		logs,
		revisions,
		remaining,
		writes: [] as { method: string; path: string; body: Record<string, any> }[],
		receipts: new Map<string, Record<string, unknown>>(),
		sockets: [] as WebSocketRoute[],
		delayRead: null as null | { identifier: string; ready: () => void; wait: Promise<void> },
		refresh() {
			for (const socket of state.sockets)
				socket.send(JSON.stringify({ type: 'app.refresh', payload: { resources: ['workspace', 'issues'], privacy: true } }));
		}
	};
	async function attach(page: Page, view: 'page' | 'panel' = 'page') {
		await page.addInitScript(() => localStorage.setItem('PARAGLIDE_LOCALE', 'en'));
		await page.routeWebSocket('**/ws', (socket) => {
			state.sockets.push(socket);
		});
		await page.route('https://raw.githubusercontent.com/carbogninalberto/kuayle/main/UI/static/releases.json', (route) =>
			route.fulfill({ json: [] })
		);
		await page.route(
			(url) => url.pathname.startsWith('/api/'),
			async (route) => {
				const path = new URL(route.request().url()).pathname;
				const method = route.request().method();
				if (path === '/api/auth/me') return route.fulfill({ json: user });
				if (path === '/api/preferences')
					return route.fulfill({
						json: { theme_mode: 'system', light_theme: 'light', dark_theme: 'dark', font_size: 'default', pointer_cursors: true }
					});
				const workspace = {
					id: 'test-id',
					slug: 'test',
					name: 'Test workspace',
					current_user_role: state.role,
					owner_id: userId,
					created_at: timestamp,
					updated_at: timestamp
				};
				if (path === '/api/workspaces') return route.fulfill({ json: [workspace] });
				if (path === '/api/workspaces/test') return route.fulfill({ json: workspace });
				if (path.endsWith('/efforts/settings'))
					return route.fulfill({ json: { enabled: state.enabled, timezone: 'Pacific/Kiritimati', revision: 0, today: state.today } });
				if (path.endsWith('/efforts/worklog-owners')) return route.fulfill({ json: [person(), person(memberId)] });
				const match = path.match(/\/issues\/(EP-[12])\/efforts(?:\/worklogs(?:\/([^/]+))?)?$/);
				if (match) {
					const identifier = match[1];
					if (!path.includes('/worklogs'))
						return route.fulfill({
							json: {
								revision: revisions[identifier],
								own: {
									estimate_minutes: 480,
									spent_minutes: logs[identifier].reduce((sum, row) => sum + row.duration_minutes, 0),
									remaining_minutes: remaining[identifier]
								},
								total: {
									estimate_minutes: 480,
									spent_minutes: 0,
									remaining_minutes: remaining[identifier],
									estimate_partial: false,
									remaining_partial: false
								}
							}
						});
					if (method === 'GET') {
						const pageNumber = Number(new URL(route.request().url()).searchParams.get('page') ?? 1);
						const json = {
							revision: revisions[identifier],
							data: structuredClone(logs[identifier].slice((pageNumber - 1) * 50, pageNumber * 50)),
							total_count: logs[identifier].length,
							page: pageNumber,
							per_page: 50,
							has_more: pageNumber * 50 < logs[identifier].length
						};
						const delay = state.delayRead;
						if (delay?.identifier === identifier) {
							state.delayRead = null;
							delay.ready();
							await delay.wait;
						}
						return route.fulfill({ json });
					}
					const body = route.request().postDataJSON();
					state.writes.push({ method, path, body });
					if (state.writeStatus)
						return route.fulfill({ status: state.writeStatus, json: { error: { code: state.writeStatus === 403 ? 'FORBIDDEN' : 'NOT_FOUND' } } });
					if (state.dailyLimit) return route.fulfill({ status: 400, json: { error: { code: 'EFFORT_DAILY_LIMIT' } } });
					let receipt = state.receipts.get(body.request_id);
					if (!receipt) {
						if (state.conflict) {
							state.conflict = false;
							revisions[identifier]++;
						}
						if (body.expected_revision !== revisions[identifier])
							return route.fulfill({ status: 409, json: { error: { code: 'EFFORT_REVISION_CONFLICT' } } });
						if (!state.enabled) return route.fulfill({ status: 409, json: { error: { code: 'EFFORT_DISABLED' } } });
						let id = match[2];
						if (method === 'POST') {
							id = crypto.randomUUID();
							logs[identifier].unshift({
								...makeLog(1, identifier),
								id,
								owner: person(body.owner_user_id),
								description: body.description,
								work_date: body.work_date,
								duration_minutes: body.duration_minutes
							});
							remaining[identifier] = Math.max(remaining[identifier] - body.duration_minutes, 0);
						} else {
							const index = logs[identifier].findIndex((row) => row.id === id);
							if (index < 0) return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND' } } });
							if (method === 'DELETE') logs[identifier].splice(index, 1);
							else
								Object.assign(logs[identifier][index], {
									work_date: body.work_date,
									duration_minutes: body.duration_minutes,
									description: body.description
								});
						}
						receipt = { request_id: body.request_id, resource_id: id, revision: ++revisions[identifier] };
						state.receipts.set(body.request_id, receipt);
					}
					if (state.loseResponse) {
						state.loseResponse = false;
						return route.abort('failed');
					}
					return route.fulfill({ json: receipt });
				}
				const issueMatch = path.match(/\/issues\/(EP-[12])$/);
				if (issueMatch) return route.fulfill({ json: issues.find((issue) => issue.identifier === issueMatch[1]) });
				if (path.endsWith('/issues')) return route.fulfill({ json: { data: issues, total_count: 2, page: 1, per_page: 50, has_more: false } });
				const project = {
					id: projectId,
					name: 'Worklog project',
					team_id: teamId,
					status: 'planned',
					start_date: null,
					target_date: null,
					description: null,
					created_at: timestamp,
					updated_at: timestamp
				};
				if (path.endsWith(`/projects/${projectId}`)) return route.fulfill({ json: project });
				if (path.endsWith('/projects')) return route.fulfill({ json: [project] });
				if (path.endsWith('/teams'))
					return route.fulfill({
						json: [{ id: teamId, name: 'Engineering', key: 'EP', color: '#6366f1', icon: 'layers', created_at: timestamp, updated_at: timestamp }]
					});
				if (path.endsWith('/statuses'))
					return route.fulfill({ json: [{ ...issues[0].status_info, team_id: teamId, slug: 'todo', is_default: true }] });
				if (path.endsWith('/github') || path.endsWith('/github/status'))
					return route.fulfill({ json: { connected: false, repos: [], pull_requests: [], branches: [], commits: [] } });
				if (path.endsWith('/dev-machine-scope-setting')) return route.fulfill({ json: { github_repo_id: null, environment_id: null } });
				if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
				if (
					['/comments', '/history', '/relations', '/sub-issues', '/members', '/labels', '/views', '/cycles', '/dev-machine-environments'].some(
						(suffix) => path.endsWith(suffix)
					)
				)
					return route.fulfill({ json: [] });
				return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND', message: path } } });
			}
		);
		await page.goto(view === 'panel' ? `/test/projects/${projectId}` : '/test/issue/EP-1');
		if (view === 'panel') await page.getByText('Worklog issue 1', { exact: true }).click();
		await expect(register(page)).toBeVisible();
		await expect(register(page)).toHaveAttribute('aria-busy', 'false');
	}
	return { state, attach };
}

async function openCreate(page: Page) {
	await register(page).getByRole('button', { name: 'Log work', exact: true }).click();
	await expect(dialog(page).getByLabel('Duration', { exact: true })).toBeEnabled();
}
async function fill(page: Page, duration = '2h', description = 'Verification\n<script>literal text</script>') {
	await dialog(page).getByLabel('Duration', { exact: true }).fill(duration);
	await dialog(page).getByLabel('Description (optional)', { exact: true }).fill(description);
}

for (const view of ['page', 'panel'] as const) {
	test(`member creates, corrects and deletes work in the ${view}`, async ({ page }, testInfo) => {
		const { state, attach } = fixture();
		await attach(page, view);
		await openCreate(page);
		await expect(dialog(page).getByLabel('Work date', { exact: true })).toHaveValue('2026-01-02');
		await fill(page);
		await dialog(page).screenshot({ path: testInfo.outputPath('worklog-dialog.png') });
		await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
		await expect(dialog(page)).toHaveCount(0);
		await expect(rows(page)).toHaveCount(1);
		await expect(rows(page)).toContainText('2h');
		await expect(rows(page)).toContainText('<script>literal text</script>');
		await register(page).screenshot({ path: testInfo.outputPath('worklog-created.png') });
		await expect(page.getByLabel('Time spent', { exact: true })).toHaveText('2h');
		expect(state.remaining['EP-1']).toBe(360);
		await rows(page).getByRole('button', { name: 'Edit work log' }).click();
		await expect(dialog(page).getByLabel('Duration', { exact: true })).toBeEnabled();
		await fill(page, '3h', 'Corrected');
		await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
		await expect(rows(page)).toContainText('3h');
		expect(state.remaining['EP-1']).toBe(360);
		await rows(page).getByRole('button', { name: 'Delete', exact: true }).click();
		await dialog(page).getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(rows(page)).toHaveCount(1);
		await rows(page).getByRole('button', { name: 'Delete', exact: true }).click();
		await expect(dialog(page).getByRole('button', { name: 'Delete', exact: true })).toBeEnabled();
		await dialog(page).getByRole('button', { name: 'Delete', exact: true }).click();
		await expect(rows(page)).toHaveCount(0);
		expect(state.remaining['EP-1']).toBe(360);
		await register(page).screenshot({ path: testInfo.outputPath('worklog-register.png') });
	});
}

test('admin delegates work and cannot reassign its owner during correction', async ({ page }) => {
	const { state, attach } = fixture('admin');
	await attach(page);
	await openCreate(page);
	await dialog(page).getByLabel('Work owner', { exact: true }).selectOption(memberId);
	await fill(page);
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(rows(page)).toContainText('Other Member');
	await expect(rows(page)).toContainText('Entered by Test User');
	await rows(page).getByRole('button', { name: 'Edit work log' }).click();
	await expect(dialog(page).getByRole('combobox')).toHaveCount(0);
	expect(state.logs['EP-1'][0].owner.id).toBe(memberId);
});

for (const mode of ['create', 'update', 'delete'] as const) {
	test(`lost ${mode} response retries exactly the same submission`, async ({ page }) => {
		const { state, attach } = fixture('member', mode === 'create' ? 0 : 1);
		await attach(page);
		if (mode === 'create') {
			await openCreate(page);
			await fill(page, '60', 'Retry me');
		} else {
			await rows(page)
				.getByRole('button', { name: mode === 'update' ? 'Edit work log' : 'Delete', exact: true })
				.click();
			await expect(dialog(page).getByRole('button', { name: mode === 'delete' ? 'Delete' : 'Save', exact: true })).toBeEnabled();
		}
		state.loseResponse = true;
		await dialog(page)
			.getByRole('button', { name: mode === 'delete' ? 'Delete' : 'Save', exact: true })
			.click();
		await expect(dialog(page).getByRole('alert')).toContainText('could not be confirmed');
		state.enabled = false;
		state.refresh();
		await expect(register(page)).toContainText('Efforts is disabled');
		await expect(dialog(page).getByRole('button', { name: 'Retry save', exact: true })).toBeEnabled();
		await dialog(page).getByRole('button', { name: 'Retry save', exact: true }).click();
		await expect(dialog(page)).toHaveCount(0);
		expect(state.writes).toHaveLength(2);
		expect(state.writes[1]).toEqual(state.writes[0]);
		expect(state.receipts.size).toBe(1);
		expect(state.logs['EP-1']).toHaveLength(mode === 'delete' ? 0 : 1);
	});
}

test('remote changes preserve an open draft and require explicit realignment', async ({ page }) => {
	const { state, attach } = fixture('member', 1);
	await attach(page);
	await rows(page).getByRole('button', { name: 'Edit work log' }).click();
	await expect(dialog(page).getByLabel('Duration', { exact: true })).toBeEnabled();
	await fill(page, '2h', 'My draft');
	state.logs['EP-1'][0].description = 'Remote correction';
	state.revisions['EP-1']++;
	state.refresh();
	await expect(rows(page)).toContainText('Remote correction');
	await expect(dialog(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('My draft');
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog(page).getByRole('alert')).toContainText('Effort has changed');
	expect(state.writes[0].body.expected_revision).toBe(0);
	await expect(dialog(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('My draft');
	await dialog(page).getByRole('button', { name: 'Replace draft with saved values', exact: true }).click();
	await expect(dialog(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('Remote correction');
});

test('validation preserves Unicode and reports the daily limit on duration', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await openCreate(page);
	await fill(page, '0');
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog(page).getByRole('alert')).toContainText('1 to 1,440');
	expect(state.writes).toHaveLength(0);
	await fill(page, '1', '🙂'.repeat(2001));
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('🙂'.repeat(2001));
	await expect(dialog(page).getByRole('alert')).toContainText('2,000');
	await fill(page, '1', '🙂'.repeat(2000));
	state.dailyLimit = true;
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog(page).getByRole('alert')).toContainText('daily limit');
	state.dailyLimit = false;
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(rows(page)).toHaveCount(1);
	expect(state.logs['EP-1'][0].description).toBe('🙂'.repeat(2000));
});

test('pagination exposes the records after fifty and supports keyboard cancellation', async ({ page }) => {
	const { attach } = fixture('member', 55);
	await attach(page);
	await expect(rows(page)).toHaveCount(3);
	await register(page).getByRole('button', { name: 'Show 47 earlier entries', exact: true }).click();
	await expect(rows(page)).toHaveCount(50);
	await register(page).getByRole('button', { name: 'Next', exact: true }).click();
	await expect(rows(page)).toHaveCount(3);
	await register(page).getByRole('button', { name: 'Show 2 earlier entries', exact: true }).click();
	await expect(rows(page)).toHaveCount(5);
	await expect(rows(page).first()).toContainText('Entry 51');
	await register(page).getByRole('button', { name: 'Previous', exact: true }).click();
	await register(page).getByRole('button', { name: 'Show 47 earlier entries', exact: true }).click();
	await expect(rows(page)).toHaveCount(50);
	await openCreate(page);
	await dialog(page).getByLabel('Duration', { exact: true }).press('Escape');
	await expect(dialog(page)).toHaveCount(0);
	await expect(register(page).getByRole('button', { name: 'Log work', exact: true })).toBeFocused();
});

for (const role of ['guest', 'disabled'] as const) {
	test(`${role} keeps saved work readable and removes writing actions`, async ({ page }) => {
		const { state, attach } = fixture(role === 'guest' ? 'guest' : 'owner', 1);
		if (role === 'disabled') state.enabled = false;
		await attach(page);
		await expect(rows(page)).toHaveCount(1);
		await expect(register(page).getByRole('button', { name: 'Log work', exact: true })).toHaveCount(0);
		await expect(rows(page).getByRole('button')).toHaveCount(0);
	});
}

test('a denied submission keeps the draft and disables further writes', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await openCreate(page);
	await fill(page, '2h', 'Keep this');
	state.writeStatus = 403;
	await dialog(page).getByRole('button', { name: 'Save', exact: true }).click();
	await expect(dialog(page).getByRole('alert')).toContainText('no longer allowed');
	await expect(dialog(page).getByLabel('Description (optional)', { exact: true })).toHaveValue('Keep this');
	await expect(dialog(page).getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
});

test('a delayed register response cannot replace the latest read', async ({ page }) => {
	const { state, attach } = fixture('member', 1);
	await attach(page);
	let release!: () => void;
	let ready!: () => void;
	const reached = new Promise<void>((resolve) => (ready = resolve));
	state.delayRead = { identifier: 'EP-1', ready, wait: new Promise<void>((resolve) => (release = resolve)) };
	state.refresh();
	await reached;
	state.logs['EP-1'][0].description = 'Latest register';
	state.revisions['EP-1']++;
	state.refresh();
	await expect(rows(page)).toContainText('Latest register');
	release();
	await expect(rows(page)).toContainText('Latest register');
});

test('responses for a previous issue cannot populate the new register', async ({ page }) => {
	const { state, attach } = fixture('member', 1);
	await attach(page);
	let release!: () => void;
	let ready!: () => void;
	const reached = new Promise<void>((resolve) => (ready = resolve));
	state.delayRead = { identifier: 'EP-1', ready, wait: new Promise<void>((resolve) => (release = resolve)) };
	state.refresh();
	await reached;
	await page.goto('/test/issue/EP-2');
	await expect(register(page)).toHaveAttribute('aria-busy', 'false');
	release();
	await expect(rows(page)).toHaveCount(0);
	await expect(register(page)).not.toContainText('Entry 1');
});

test('creation rereads workspace today when the dialog opens', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	state.today = '2026-01-03';
	await openCreate(page);
	await expect(dialog(page).getByLabel('Work date', { exact: true })).toHaveValue('2026-01-03');
});

for (const view of ['page', 'panel'] as const) {
	test(`work log follows Activity and expands as a compact timeline in the ${view}`, async ({ page }, testInfo) => {
		const { attach } = fixture('member', 5);
		await attach(page, view);
		if (view === 'page') {
			await expect(register(page).locator('xpath=preceding-sibling::*[1]').getByRole('heading', { name: 'Activity', exact: true })).toBeVisible();
		} else {
			await page.getByRole('button', { name: 'Activity (0)', exact: true }).click();
			await expect(register(page).locator('xpath=preceding-sibling::*[1]')).toHaveText('No activity yet');
		}
		await expect(rows(page)).toHaveCount(3);
		const toggle = register(page).getByRole('button', { name: 'Work log', exact: true });
		await expect(toggle).toHaveAttribute('aria-expanded', 'true');
		await toggle.focus();
		await toggle.press('Enter');
		await expect(toggle).toHaveAttribute('aria-expanded', 'false');
		await expect(register(page).getByRole('list')).toBeHidden();
		await toggle.press('Space');
		await expect(register(page).getByRole('list')).toBeVisible();
		await register(page).getByRole('button', { name: 'Show 2 earlier entries', exact: true }).click();
		await expect(rows(page)).toHaveCount(5);
		await register(page).getByRole('button', { name: 'Show less', exact: true }).click();
		await expect(rows(page)).toHaveCount(3);
		await register(page).screenshot({ path: testInfo.outputPath('worklog-timeline.png'), animations: 'disabled' });
		await page.emulateMedia({ colorScheme: 'dark' });
		await expect(page.locator('html')).toHaveClass(/dark/);
		await register(page).screenshot({ path: testInfo.outputPath('worklog-timeline-dark.png'), animations: 'disabled' });
		if (view === 'page') {
			await page
				.getByRole('heading', { name: 'Activity', exact: true })
				.locator('..')
				.screenshot({ path: testInfo.outputPath('activity-reference.png'), animations: 'disabled' });
		}
		await page.setViewportSize({ width: 390, height: 844 });
		await register(page).scrollIntoViewIfNeeded();
		await register(page).screenshot({ path: testInfo.outputPath('worklog-timeline-mobile.png'), animations: 'disabled' });
		await toggle.click();
		await register(page).getByRole('button', { name: 'Log work', exact: true }).click();
		await expect(dialog(page).getByLabel('Duration', { exact: true })).toBeEnabled();
		await dialog(page).getByRole('button', { name: 'Cancel', exact: true }).click();
		await expect(toggle).toHaveAttribute('aria-expanded', 'true');
	});
}
