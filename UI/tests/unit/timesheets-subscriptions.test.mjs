import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';
const source = await readFile(new URL('../../src/lib/features/timesheets/subscriptions.ts', import.meta.url), 'utf8');
const { createTimesheetSubscriptions } = await import(`data:text/javascript;base64,${Buffer.from(stripTypeScriptTypes(source)).toString('base64')}`);
function fixture() {
 const sockets = []; const statuses = []; let invalidations = 0;
 const subscriptions = createTimesheetSubscriptions(() => invalidations++, status => statuses.push(status), slug => {
  const socket = { slug, closed: false, close() { this.closed = true; this.onclose?.(); } };
  sockets.push(socket); return socket;
 });
 return { subscriptions, sockets, statuses, count: () => invalidations };
}
test('selection retains sockets and ignores removed or previous-user frames', () => {
 const f = fixture(); f.subscriptions.update('one', ['a', 'b', 'a']);
 assert.equal(f.sockets.length, 2);
 f.sockets.forEach(s => s.onopen()); assert.equal(f.count(), 2);
 assert.equal(f.statuses.at(-1), true);
 f.subscriptions.update('one', ['b']); assert.equal(f.sockets[0].closed, true);
 f.sockets[0].onmessage({ data: '{"type":"app.refresh","payload":{"resources":["timesheets"]}}' });
 assert.equal(f.count(), 2);
 f.subscriptions.update('two', ['b']); assert.equal(f.sockets[1].closed, true);
 assert.equal(f.sockets.length, 3);
 f.sockets[1].onopen(); assert.equal(f.count(), 2);
 f.subscriptions.update(undefined, ['b']); assert.equal(f.sockets[2].closed, true);
 f.subscriptions.close();
});
test('only valid invalidations reload; reconnect refreshes and cleanup cancels retries', t => {
 t.mock.timers.enable({ apis: ['setTimeout'] });
 const f = fixture(); f.subscriptions.update('one', ['a']); const first = f.sockets[0];
 for (const data of ['bad', '{"type":"presence.join"}', '{"type":"app.refresh","payload":{"resources":["issues"]}}']) first.onmessage({ data });
 assert.equal(f.count(), 0);
 first.onmessage({ data: '{"type":"app.refresh","payload":{"resources":["timesheets"]}}' });
 assert.equal(f.count(), 1);
 first.onclose(); assert.equal(f.count(), 2); assert.equal(f.statuses.at(-1), false);
 t.mock.timers.tick(2999); assert.equal(f.sockets.length, 1);
 t.mock.timers.tick(1); assert.equal(f.sockets.length, 2);
 f.sockets[1].onopen(); assert.equal(f.count(), 3);
 first.onopen(); assert.equal(f.count(), 3);
 f.sockets[1].onclose(); f.subscriptions.close();
 t.mock.timers.tick(3000); assert.equal(f.sockets.length, 2);
});
