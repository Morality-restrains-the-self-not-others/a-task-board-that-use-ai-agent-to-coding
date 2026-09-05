const RETURN_URL_MAX_LEN = 2048

export class PostLoginReturnUrl {
  constructor(raw) {
    const normalized = PostLoginReturnUrl.normalize(raw)
    if (!normalized) {
      throw new Error('无效的登录回跳地址')
    }
    this._value = normalized
  }

  static normalize(raw) {
    if (raw == null || typeof raw !== 'string') return null
    const trimmed = raw.trim()
    if (!trimmed || trimmed.length > RETURN_URL_MAX_LEN) return null
    if (!trimmed.startsWith('/')) return null
    if (trimmed.startsWith('//')) return null
    if (trimmed.includes('://')) return null
    if (/[\n\r\0]/.test(trimmed)) return null
    return trimmed
  }

  get value() {
    return this._value
  }
}
