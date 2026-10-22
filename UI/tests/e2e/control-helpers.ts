import { expect, type Locator } from '@playwright/test';

export function dateField(root: Locator, label: string) {
	return root.locator('[data-date-field]').filter({ has: root.page().getByRole('group', { name: label, exact: true }) });
}

export async function fillDate(root: Locator, label: string, value: string) {
	const [year, month, day] = value.split('-');
	const field = dateField(root, label);
	for (const [part, text] of [['year', year], ['month', month], ['day', day]]) {
		const segment = field.locator(`[data-segment="${part}"]`);
		await segment.focus();
		await segment.pressSequentially(text);
	}
	await expect(field).toHaveAttribute('data-value', value);
}

export async function chooseForm(root: Locator, label: string, option: string) {
	await root.getByLabel(label, { exact: true }).click();
	await root.page().getByRole('option', { name: option, exact: true }).click();
}

export async function worklogAction(root: Locator, action: string) {
	await root.getByRole('button', { name: 'Worklog actions', exact: true }).click();
	await root.page().getByRole('menuitem', { name: action, exact: true }).click();
}
