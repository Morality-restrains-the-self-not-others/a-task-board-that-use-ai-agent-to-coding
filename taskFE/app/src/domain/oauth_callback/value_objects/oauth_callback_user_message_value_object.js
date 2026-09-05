export class OAuthCallbackUserMessage {
  constructor(raw) {
    const value = String(raw ?? '').trim()
    if (!value) {
      throw new Error('OAuth 回调展示文案不能为空')
    }
    this._value = value
  }

  get value() {
    return this._value
  }
}
