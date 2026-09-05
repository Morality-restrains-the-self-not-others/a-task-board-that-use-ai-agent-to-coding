const ALLOWED = new Set(['success', 'error'])

export class OAuthCallbackSeverity {
  constructor(raw) {
    const value = String(raw ?? '').trim().toLowerCase()
    if (!ALLOWED.has(value)) {
      throw new Error(`无效的回调严重级别: ${raw}`)
    }
    this._value = value
  }

  get value() {
    return this._value
  }

  isError() {
    return this._value === 'error'
  }

  isSuccess() {
    return this._value === 'success'
  }
}
