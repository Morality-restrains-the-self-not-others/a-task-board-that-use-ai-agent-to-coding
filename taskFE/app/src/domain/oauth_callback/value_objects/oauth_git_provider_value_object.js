const ALLOWED = new Set(['github', 'gitlab'])

export class OAuthGitProvider {
  constructor(raw) {
    const value = String(raw ?? '').trim().toLowerCase()
    if (!ALLOWED.has(value)) {
      throw new Error(`无效的 OAuth provider: ${raw}`)
    }
    this._value = value
  }

  get value() {
    return this._value
  }

  /** URL query 键名，与 gitOauth 回调契约一致 */
  get queryKey() {
    return this._value
  }

  equals(other) {
    return other instanceof OAuthGitProvider && other._value === this._value
  }
}
