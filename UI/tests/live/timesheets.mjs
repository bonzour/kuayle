import { chromium } from 'playwright';
import { expect } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
// Opt-in, isolated migrated PostgreSQL/backend, same-origin HTTP + WS proxy.
// No HTTP routes or WebSocket frames are mocked. Drop the test database after use.
const baseURL = process.env.KUAYLE_LIVE_URL ?? 'http://127.0.0.1:4176';
const evidenceDir = fileURLToPath(new URL('../../../tmp/reports/ep06/', import.meta.url));
await mkdir(evidenceDir, { recursive: true });
const browser = await chromium.launch({ headless: true });
const owner = await browser.newContext({ baseURL });
const actor = await browser.newContext({ baseURL, viewport: { width: 1440, height: 1000 } });
const page = await actor.newPage();
const frames = [], errors = [], workspaces = [];
const suffix = Date.now();
page.on('pageerror', error => errors.push(error.message));
page.on('websocket', socket => {
 if (!socket.url().includes('channel=timesheets')) return;
 socket.on('framereceived', frame => frames.push({ url: socket.url(), data: String(frame.payload) }));
});
await page.addInitScript(() => {
 localStorage.setItem('PARAGLIDE_LOCALE', 'en');
 window.__timesheetSockets = [];
 const NativeWebSocket = window.WebSocket;
 window.WebSocket = class extends NativeWebSocket {
  constructor(...args) { super(...args); if (String(args[0]).includes('channel=timesheets')) window.__timesheetSockets.push(this); }
 };
});
async function request(context, method, path, data, expected) {
 let response;
 try { response = await context.request.fetch(path, { method, data }); }
 catch { throw new Error(`${method} ${path}: connection failed`); }
 if (expected) expect(response.status()).toBe(expected);
 else if (!response.ok()) throw new Error(`${method} ${path}: ${response.status()} ${await response.text()}`);
 const text = await response.text(); return text ? JSON.parse(text) : null;
}
const form = () => page.getByRole('dialog', { name: /^(Log work|Edit work log)$/ });
try {
 const ownerUser = await request(owner, 'POST', '/api/auth/register', { name: 'EP06 Owner', email: `ep06-owner-${suffix}@test.invalid`, password: 'EP06-isolated-password!' });
 const actorUser = await request(actor, 'POST', '/api/auth/register', { name: 'EP06 Reader', email: `ep06-reader-${suffix}@test.invalid`, password: 'EP06-isolated-password!' });
 const issues = [];
 for (const [index, role] of ['admin', 'member', 'guest'].entries()) {
  const workspace = await request(owner, 'POST', '/api/workspaces', { name: `EP06 ${['A','B','C'][index]}`, slug: `ep06-${suffix}-${index}` });
  workspaces.push(workspace);
  const root = `/api/workspaces/${workspace.slug}`;
  await request(owner, 'POST', `${root}/invite`, { email: actorUser.email, role });
  const team = await request(owner, 'POST', `${root}/teams`, { name: 'EP06 Team', key: ['EPA', 'EPB', 'EPC'][index] });
  issues.push(await request(owner, 'POST', `${root}/issues`, { team_id: team.id, title: `EP06 work ${index}` }));
 }
 const [a,b] = workspaces;
 const log = async (index, minutes, revision) => request(owner, 'POST', `/api/workspaces/${workspaces[index].slug}/issues/${issues[index].identifier}/efforts/worklogs`, { request_id: crypto.randomUUID(), expected_revision: revision, owner_user_id: actorUser.id, work_date: '2025-10-06', end_date: '2025-10-06', duration_minutes: minutes, description: 'Remote EP06 work' });
 await log(0,60,0); await log(1,30,0);
 const query = new URLSearchParams({ period: 'week', from: '2025-10-06', to: '2025-10-12', reference_workspace_id: a.id });
 await page.goto(`/${a.slug}/timesheet?${query}`);
 const cell = value => page.getByRole('button', { name: `${issues[1].identifier} · 2025-10-06 · ${value}`, exact: true });
 await expect(cell('30m')).toBeVisible({ timeout: 30000 });
 await expect.poll(() => page.evaluate(() => window.__timesheetSockets.filter(s => s.readyState === WebSocket.OPEN).length)).toBe(3);
 await log(1,45,1);
 await expect(cell('1h 15m')).toBeVisible();
 console.log('PASS remote B work updates Timesheet opened from A using real HTTP and WebSocket');
 await cell('1h 15m').click();
 const detail = page.getByRole('dialog', { name: 'Worklog details', exact: true });
 await expect(detail).toContainText('2 registrations');
 await detail.getByRole('button', { name: 'Log work', exact: true }).click();
 await form().getByLabel('Duration', { exact: true }).fill('2h');
 await form().getByLabel('Description (optional)', { exact: true }).fill('Live draft retained');
 await log(1,15,2);
 await expect(page.locator(`button[aria-label="${issues[1].identifier} · 2025-10-06 · 1h 30m"]`)).toHaveCount(1);
 await expect(form().getByLabel('Description (optional)', { exact: true })).toHaveValue('Live draft retained');
 console.log('PASS remote refresh preserves an open worklog draft and refreshes detail');
 await page.evaluate(slug => window.__timesheetSockets.find(s => s.url.includes(`/workspaces/${slug}/`) && s.readyState === WebSocket.OPEN).close(), b.slug);
 await log(1,15,3);
 await expect.poll(() => page.evaluate(slug => window.__timesheetSockets.filter(s => s.url.includes(`/workspaces/${slug}/`)).length, b.slug)).toBe(2);
 await expect(page.locator(`button[aria-label="${issues[1].identifier} · 2025-10-06 · 1h 45m"]`)).toHaveCount(1);
 await expect(form().getByLabel('Duration', { exact: true })).toHaveValue('2h');
 console.log('PASS reconnect rereads remote work committed during disconnection');
 await request(owner, 'DELETE', `/api/workspaces/${b.slug}/members/${actorUser.id}`);
 await expect(page.locator(`button[aria-label^="${issues[1].identifier} ·"]`)).toHaveCount(0, { timeout: 15000 });
 await expect(form().getByLabel('Description (optional)', { exact: true })).toHaveValue('Live draft retained');
 await form().getByRole('button', { name: 'Save', exact: true }).click();
 await expect(form()).toBeVisible();
 await expect(form().getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
 expect(new URL(page.url()).searchParams.get('from')).toBe('2025-10-06');
 console.log('PASS revocation clears B, preserves dates and draft, and prevents saving');
 await form().getByRole('button', { name: 'Cancel', exact: true }).click();
 for (const frame of frames) expect(JSON.parse(frame.data)).toEqual({ type: 'app.refresh', payload: { resources: ['timesheets'] } });
 expect(frames.length).toBeGreaterThan(0); expect(errors).toEqual([]);
 await page.setViewportSize({ width: 390, height: 844 });
 expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
 await page.screenshot({ path: `${evidenceDir}/live-revoked-mobile.png`, fullPage: true });
 await writeFile(`${evidenceDir}/live-evidence.json`, JSON.stringify({ timestamp: new Date().toISOString(), workspaces: workspaces.map(w => w.id), owner: ownerUser.id, actor: actorUser.id, frames, pageErrors: errors }, null, 2));
 console.log('PASS invalidation-only frames, mobile viewport, zero page errors');
} catch (error) {
 await page.screenshot({ path: `${evidenceDir}/live-failure.png`, fullPage: true });
 await writeFile(`${evidenceDir}/live-failure.json`, JSON.stringify({ frames, pageErrors: errors }, null, 2));
 throw error;
} finally {
 try {
  for (const workspace of workspaces.reverse()) await request(owner, 'DELETE', `/api/workspaces/${workspace.slug}`);
 } finally { await browser.close(); }
}
