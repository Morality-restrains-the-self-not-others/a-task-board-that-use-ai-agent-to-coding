/** 个人资料页手机绑定/换绑 API 契约（Django 遗留路径，由 taskAuth 承接）。 */

export const PROFILE_BIND_PHONE_PATH = '/api/accounts/users/profile/bind-phone/'
export const PROFILE_REPLACE_PHONE_PATH = '/api/accounts/users/profile/replace-phone/'

export const PHONE_TAKEN_ERROR_CODE = 'phone_taken'
export const PHONE_BIND_LIMIT_ERROR_CODE = 'phone_bind_limit'
export const PHONE_AMBIGUOUS_ERROR_CODE = 'phone_ambiguous'

/** 仅当服务端确认 bound 才视为绑定成功（避免 profile upsert 200/{ok:true} 误跳转）。 */
export function shouldNavigateAfterPhoneBind(responseOk, data) {
  return Boolean(responseOk && data && data.bound === true)
}

export function isPhoneBindLimitError(data) {
  if (!data || typeof data !== 'object') return false
  return data.code === PHONE_BIND_LIMIT_ERROR_CODE
}

export function isPhoneTakenBindError(data) {
  if (!data || typeof data !== 'object') return false
  if (isPhoneBindLimitError(data)) return false
  return data.code === PHONE_TAKEN_ERROR_CODE || data.reclaim_available === true
}

export function bindPhoneRequestBody({ phone, code, reclaim = false }) {
  const body = { phone, code }
  if (reclaim) body.reclaim = true
  return body
}
