import type { Role } from './roles';

const ROLE_PERMISSIONS: Record<Role, string[]> = {
	owner: [
		'efforts:read',
		'efforts:write',
		'workspace:manage',
		'team:manage',
		'issue:create',
		'issues:read',
		'issue:update',
		'issue:delete',
		'issue:delete_own',
		'project:manage',
		'label:manage',
		'member:invite',
		'cycle:manage',
		'view:manage'
	],
	admin: [
		'efforts:read',
		'efforts:write',
		'team:manage',
		'issue:create',
		'issues:read',
		'issue:update',
		'issue:delete',
		'issue:delete_own',
		'project:manage',
		'label:manage',
		'member:invite',
		'cycle:manage',
		'view:manage'
	],
	member: [
		'efforts:read',
		'efforts:write',
		'issue:create',
		'issues:read',
		'issue:update',
		'issue:delete_own',
		'project:manage',
		'label:manage',
		'cycle:manage',
		'view:manage'
	],
	guest: ['issues:read', 'efforts:read']
};

export function hasPermission(role: Role, permission: string): boolean {
	return ROLE_PERMISSIONS[role]?.includes(permission) ?? false;
}
