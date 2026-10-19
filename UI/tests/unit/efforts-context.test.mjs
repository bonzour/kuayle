import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';
import { compileModule } from 'svelte/compiler';

// Compile the real runes module; substitute only context and HTTP adapters.
async function loadContext() {
	const source = (await readFile(new URL('../../src/lib/features/workspaces/current-workspace.svelte.ts', import.meta.url), 'utf8'))
		.replace("import { getContext, setContext } from 'svelte';", 'const getContext = () => null; const setContext = (_key, value) => value;')
		.replace("import { getEffortSettings } from '$lib/api/efforts';", 'export const reads = []; const getEffortSettings = (slug) => new Promise((resolve, reject) => reads.push({ slug, resolve, reject }));');
	const compiled = compileModule(stripTypeScriptTypes(source), { filename: 'current-workspace.svelte.js', generate: 'client' }).js.code
		.replace(/(["'])(svelte\/[^"']+)\1/g, (_match, quote, specifier) => `${quote}${import.meta.resolve(specifier)}${quote}`);
	return import(`data:text/javascript;base64,${Buffer.from(`${compiled}\n// ${Math.random()}`).toString('base64')}`);
}
const membership = (slug = 'test', userId = 'user-1') => ({ workspace: { slug }, userId });
const settings = (revision) => ({ enabled: true, timezone: 'UTC', today: '2026-10-17', revision });

test('a slower read cannot overwrite the latest settings response', async () => {
	const { setCurrentWorkspace, reads } = await loadContext();
	const context = setCurrentWorkspace();
	context.membership = membership();
	const first = context.loadEfforts('test');
	const second = context.loadEfforts('test');
	reads[1].resolve(settings(2));
	await second;
	reads[0].resolve(settings(1));
	assert.equal(await first, null);
	assert.equal(context.efforts.settings.revision, 2);
});

test('workspace switches discard old responses and ignore stale callback reloads', async () => {
	const { setCurrentWorkspace, reads } = await loadContext();
	const context = setCurrentWorkspace();
	context.membership = membership();
	const oldRead = context.loadEfforts('test');
	context.clearEfforts();
	context.membership = membership('other');
	const currentRead = context.loadEfforts('other');
	assert.equal(await context.loadEfforts('test'), null);
	assert.equal(reads.length, 2, 'a stale callback must not issue a read or supersede the new workspace');
	reads[0].resolve(settings(1));
	await oldRead;
	reads[1].resolve(settings(2));
	await currentRead;
	assert.equal(context.efforts.slug, 'other');
	assert.equal(context.efforts.settings.revision, 2);
});

test('responses for a previous user cannot populate the current workspace', async () => {
	const { setCurrentWorkspace, reads } = await loadContext();
	const context = setCurrentWorkspace();
	context.membership = membership();
	const pending = context.loadEfforts('test');
	context.membership = membership('test', 'user-2');
	reads[0].resolve(settings(1));
	assert.equal(await pending, null);
	assert.equal(context.efforts, null);
});

test('a superseded read failure cannot clear newer settings', async () => {
	const { setCurrentWorkspace, reads } = await loadContext();
	const context = setCurrentWorkspace();
	context.membership = membership();
	const first = context.loadEfforts('test');
	const second = context.loadEfforts('test');
	reads[1].resolve(settings(2));
	await second;
	reads[0].reject(new Error('connection lost'));
	await first;
	assert.equal(context.efforts.settings.revision, 2);
	assert.equal(context.effortsFailed, false);
});
