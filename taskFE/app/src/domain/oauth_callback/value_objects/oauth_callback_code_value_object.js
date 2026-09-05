export class OAuthCallbackCode {
  constructor(raw) {
    const value = String(raw ?? '').trim()
    if (!value) {
      throw new Error('OAuth 回调码不能为空')
    }
    this._value = value
  }

  get value() {
    return this._value
  }

  isSuccess() {
    return this._value === 'ok'
  }

  equals(other) {
    return other instanceof OAuthCallbackCode && other._value === this._value
  }
}
