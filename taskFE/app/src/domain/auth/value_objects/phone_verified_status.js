/**
 * 登录/资料载荷上的「已验证手机号」投影。
 * 绑定接口仅在短信验证成功后写入 is_verified=1 的 phone login_method。
 */
export function userHasVerifiedPhone(source) {
  if (!source || typeof source !== 'object') return false
  const methods = source.login_methods
  if (Array.isArray(methods)) {
    return methods.some(
      (m) => m && m.method_type === 'phone' && m.is_verified === true,
    )
  }
  if (typeof source.has_phone === 'boolean') {
    return source.has_phone === true
  }
  return false
}
