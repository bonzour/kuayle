import { test, expect, type Page, type WebSocketRoute } from '@playwright/test';

const initial = () => ({ enabled: true, timezone: 'Europe/Rome', revision: 0, today: '2026-10-17' });
const userId = '00000000-0000-0000-0000-000000000001';
const panel = (page: Page) => page.getByRole('region', { name: 'Efforts', exact: true });
const zone = (page: Page) => panel(page).getByLabel('Workspace time zone', { exact: true });
const save = (page: Page) => panel(page).getByRole('button', { name: 'Save', exact: true });

function fixture(role = 'owner') {
	const state = {
		role, settings: initial(), writes: [] as Record<string, unknown>[],
		receipts: new Map<string, Record<string, unknown>>(),
		loseResponse: false, conflict: false, failRead: false,
		sockets: [] as WebSocketRoute[],
		refresh() {
			for (const socket of state.sockets) socket.send(JSON.stringify({ type: 'app.refresh', payload: { resources: ['workspace', 'issues'], privacy: true } }));
		}
	};
	async function attach(page: Page) {
		await page.addInitScript(() => localStorage.setItem('PARAGLIDE_LOCALE', 'en'));
		await page.routeWebSocket('**/ws', socket => { state.sockets.push(socket); });
		await page.route('https://raw.githubusercontent.com/carbogninalberto/kuayle/main/UI/static/releases.json', route => route.fulfill({ json: [] }));
		await page.route('**/api/**', async route => {
			const request = route.request();
			const path = new URL(request.url()).pathname;
			if (path === '/api/auth/me') return route.fulfill({ json: { id: userId, email: 'test@example.com', name: 'Test User', avatar_url: null } });
			if (path === '/api/preferences') return route.fulfill({ json: {} });
			if (path === '/api/workspaces/test') return route.fulfill({ json: {
				id: 'test-id', slug: 'test', name: 'Test workspace', owner_id: role === 'owner' ? userId : 'other-user',
				current_user_role: state.role, logo_url: null, share_link_min_role: 'admin',
				created_at: '2026-10-17T12:00:00Z', updated_at: '2026-10-17T12:00:00Z'
			} });
			if (path === '/api/workspaces/test/efforts/settings') {
				if (request.method() === 'GET') return state.failRead
					? route.fulfill({ status: 500, json: { error: { code: 'INTERNAL_ERROR' } } })
					: route.fulfill({ json: state.settings });
				const body = request.postDataJSON();
				state.writes.push(body);
				if (state.conflict) {
					state.conflict = false;
					state.settings = { ...state.settings, timezone: 'UTC', revision: state.settings.revision + 1 };
					return route.fulfill({ status: 409, json: { error: { code: 'EFFORT_REVISION_CONFLICT' } } });
				}
				if (body.timezone === 'invalid') return route.fulfill({ status: 400, json: { error: { code: 'VALIDATION_ERROR', details: [{ field: 'timezone' }] } } });
				let receipt = state.receipts.get(body.request_id);
				if (!receipt) {
					state.settings = { ...state.settings, enabled: body.enabled, timezone: body.timezone, revision: state.settings.revision + 1 };
					receipt = { request_id: body.request_id, revision: state.settings.revision, resource_id: 'test-id' };
					state.receipts.set(body.request_id, receipt);
				}
				if (state.loseResponse) { state.loseResponse = false; return route.abort('failed'); }
				return route.fulfill({ json: receipt });
			}
			if (path === '/api/notifications') return route.fulfill({ json: { notifications: [], unread_count: 0 } });
			if (path.endsWith('/issues')) return route.fulfill({ json: { data: [], total_count: 0, page: 1, per_page: 50, has_more: false } });
			if (['/teams', '/members', '/projects', '/labels', '/views'].some(suffix => path.endsWith(suffix))) return route.fulfill({ json: [] });
			return route.fulfill({ status: 404, json: { error: { code: 'NOT_FOUND', message: path } } });
		});
		await page.goto('/test/settings');
		await expect(zone(page)).toHaveValue('Europe/Rome');
	}
	return { state, attach };
}

for (const role of ['owner', 'admin', 'member', 'guest']) {
	test(`Efforts settings respect the ${role} role`, async ({ page }) => {
		const { state, attach } = fixture(role);
		await attach(page);
		if (role === 'owner' || role === 'admin') {
			await expect(zone(page)).toBeEnabled();
			await panel(page).getByRole('switch').focus();
			await page.keyboard.press('Space');
			await zone(page).fill('UTC');
			await save(page).click();
			await expect.poll(() => state.settings.enabled).toBe(false);
			await expect(zone(page)).toBeEnabled();
			await expect(zone(page)).toHaveValue('UTC');
			await panel(page).getByRole('switch').click();
			await save(page).click();
			await expect.poll(() => state.settings.enabled).toBe(true);
			await expect(save(page)).toBeDisabled();
		} else {
			await expect(zone(page)).toBeDisabled();
			await expect(panel(page).getByRole('switch')).toBeDisabled();
			await expect(save(page)).toHaveCount(0);
			expect(state.writes).toHaveLength(0);
		}
	});
}

test('validation and conflicts preserve the draft until explicit realignment', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await zone(page).fill('invalid');
	await save(page).click();
	await expect(zone(page)).toHaveAttribute('aria-invalid', 'true');
	await expect(zone(page)).toHaveValue('invalid');
	await expect(panel(page).getByRole('alert')).toContainText('valid IANA');
	await zone(page).fill('Pacific/Kiritimati');
	state.conflict = true;
	await save(page).click();
	await expect(panel(page).getByRole('alert')).toContainText('saved settings have changed');
	await expect(zone(page)).toHaveValue('Pacific/Kiritimati');
	await expect(save(page)).toBeDisabled();
	await panel(page).getByRole('button', { name: 'Replace draft with saved values' }).click();
	await expect(zone(page)).toHaveValue('UTC');
	await zone(page).fill('Europe/Rome');
	await save(page).click();
	await expect.poll(() => state.writes.at(-1)?.expected_revision).toBe(1);
	expect(state.writes[0].request_id).not.toBe(state.writes[1].request_id);
});

test('lost response retries exactly the same submission without repeating its effect', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	state.loseResponse = true;
	await zone(page).fill('UTC');
	await save(page).click();
	await expect(panel(page).getByRole('alert')).toContainText('could not be confirmed');
	await expect(zone(page)).toHaveValue('UTC');
	await expect(zone(page)).toBeDisabled();
	state.refresh();
	await panel(page).getByRole('button', { name: 'Retry save' }).click();
	await expect(zone(page)).toBeEnabled();
	expect(state.writes).toHaveLength(2);
	expect(state.writes[1]).toEqual(state.writes[0]);
	expect(state.settings.revision).toBe(1);
});

test('remote sessions refresh saved values, preserve drafts and revoke controls', async ({ browser }) => {
	const { state, attach } = fixture('admin');
	const first = await browser.newContext();
	const second = await browser.newContext();
	try {
		const a = await first.newPage();
		const b = await second.newPage();
		await attach(a);
		await attach(b);
		await zone(a).fill('UTC');
		await save(a).click();
		await expect.poll(() => state.settings.revision).toBe(1);
		state.refresh();
		await expect(zone(b)).toHaveValue('UTC');
		await zone(b).fill('Pacific/Kiritimati');
		state.settings = { ...state.settings, timezone: 'Europe/Rome', revision: 2 };
		state.refresh();
		await expect(panel(b).getByRole('status')).toContainText('saved settings have changed');
		await expect(zone(b)).toHaveValue('Pacific/Kiritimati');
		state.role = 'guest';
		state.refresh();
		await expect(zone(b)).toBeDisabled();
		await expect(save(b)).toHaveCount(0);
	} finally {
		await first.close();
		await second.close();
	}
});

test('failed reload preserves a draft until explicit recovery', async ({ page }) => {
	const { state, attach } = fixture();
	await attach(page);
	await zone(page).fill('Pacific/Kiritimati');
	state.failRead = true;
	state.refresh();
	await expect(panel(page).getByRole('alert')).toContainText('Unable to load');
	state.failRead = false;
	state.settings = { ...state.settings, timezone: 'UTC', revision: 1 };
	await panel(page).getByRole('button', { name: 'Reload', exact: true }).click();
	await expect(zone(page)).toHaveValue('UTC');
});
