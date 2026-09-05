/**
 * 将注册 API 错误体映射为页面可见状态。
 * phone_register 的通用 `error`/`message` 不得只写入 email 字段：
 * 手机注册表单不渲染邮箱错误，会导致 400 被静默吞掉。
 */

const INVITE_ERR_MAP = {
  invite_code_required: '请填写邀请码',
  invite_code_invalid: '邀请码无效',
  invite_code_used: '邀请码已被使用',
  invite_code_revoked: '邀请码已撤销',
  access_code_invalid: '邀请链接无效，请联系分享者获取新的邀请链接',
  invite_gate_unavailable: '邀请验证服务暂不可用，请稍后重试',
}

function firstField(value) {
  if (Array.isArray(value) && value.length) {
    const v = value[0]
    return typeof v === 'string' ? v : String(v ?? '')
  }
  if (typeof value === 'string' && value.trim()) return value.trim()
  return ''
}

function genericMessage(data) {
  if (!data || typeof data !== 'object') return ''
  if (typeof data.message === 'string' && data.message.trim()) return data.message.trim()
  if (typeof data.detail === 'string' && data.detail.trim()) return data.detail.trim()
  if (typeof data.error === 'string' && data.error.trim() && !INVITE_ERR_MAP[data.error]) {
    return data.error.trim()
  }
  if (Array.isArray(data.non_field_errors) && data.non_field_errors.length) {
    const msg = data.non_field_errors[0]
    if (typeof msg === 'string' && msg.trim()) return msg.trim()
  }
  return ''
}

/**
 * @param {Record<string, unknown>|null|undefined} data
 * @param {{ registerType?: string, traceId?: string }} [options]
 */
export function mapRegisterApiError(data, options = {}) {
  const registerType = options.registerType === 'email' ? 'email' : 'phone'
  const traceId =
    (typeof options.traceId === 'string' && options.traceId) ||
    (typeof data?.trace_id === 'string' && data.trace_id) ||
    (typeof data?.traceId === 'string' && data.traceId) ||
    ''
  const fieldErrors = { phone: '', email: '', code: '', password: '' }

  if (data?.error && INVITE_ERR_MAP[data.error]) {
    return {
      inviteError: INVITE_ERR_MAP[data.error],
      inviteErrorTraceId: traceId,
      fieldErrors,
      submitError: '',
      submitErrorTraceId: '',
    }
  }

  fieldErrors.phone = firstField(data?.phone)
  fieldErrors.email = firstField(data?.email)
  fieldErrors.code = firstField(data?.code)
  fieldErrors.password = firstField(data?.password)

  const generic = genericMessage(data)
  const visibleField = registerType === 'phone'
    ? Boolean(fieldErrors.phone || fieldErrors.code || fieldErrors.password)
    : Boolean(fieldErrors.email || fieldErrors.password)

  let submitError = ''
  if (generic) {
    submitError = generic
    if (registerType === 'phone' && /验证码/.test(generic) && !fieldErrors.code) {
      fieldErrors.code = generic
    }
  } else if (!visibleField) {
    submitError = '注册失败，请稍后重试'
  }

  return {
    inviteError: '',
    inviteErrorTraceId: '',
    fieldErrors,
    submitError,
    submitErrorTraceId: submitError ? traceId : '',
  }
}
