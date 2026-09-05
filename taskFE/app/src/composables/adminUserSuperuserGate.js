/**
 * 系统管理用户表单：是否允许写入 is_superuser。
 * 仅 super_admin / platform:manage（系统管理员角色）。
 */
export function canSetSuperuserFlag(isPlatformRole, hasPlatformPerm) {
  return isPlatformRole('super_admin') || hasPlatformPerm('platform:manage')
}

export function omitUnauthorizedSuperuser(form, canSet) {
  const payload = { ...form }
  if (!payload.password) {
    delete payload.password
  }
  if (!canSet) {
    delete payload.is_superuser
  }
  return payload
}
