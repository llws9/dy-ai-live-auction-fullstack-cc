export const MERCHANT_ROLE = 1;
export const ADMIN_ROLE = 2;
export const ADMIN_FRONTEND_ROLES = [MERCHANT_ROLE, ADMIN_ROLE] as const;

export function roleLabel(role: number | undefined) {
  if (role === ADMIN_ROLE) return '平台管理员';
  if (role === MERCHANT_ROLE) return '商家/主播';
  return '未授权用户';
}

export function isAllowedRole(allowedRoles: readonly number[] | undefined, role: number | undefined) {
  const roles = allowedRoles ?? ADMIN_FRONTEND_ROLES;
  return role !== undefined && roles.includes(role);
}
