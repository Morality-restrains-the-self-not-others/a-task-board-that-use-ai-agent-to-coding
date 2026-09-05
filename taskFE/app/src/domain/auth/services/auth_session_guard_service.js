import { syncUserIdCookieFromProfile } from '../../../utils/sessionUserIdUtils.js'

const PROFILE_PATH = '/api/accounts/users/profile/'

export class AuthSessionGuardService {
  constructor({ apiFetch, getCookie, clearUserIdCookie, clearAuthToken }) {
    this._apiFetch = apiFetch
    this._getCookie = getCookie
    this._clearUserIdCookie = clearUserIdCookie
    this._clearAuthToken = clearAuthToken
  }

  async _meSessionValid(userId) {
    const uid = String(userId || '').trim()
    if (!uid) return false
    try {
      const meResponse = await this._apiFetch(`/api/accounts/users/me/`, {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      })
      return meResponse.ok
    } catch {
      return false
    }
  }

  async isAuthenticated() {
    const userId = String(this._getCookie('userId') || '').trim()
    if (userId && (await this._meSessionValid(userId))) {
      return true
    }

    try {
      const profileResponse = await this._apiFetch(PROFILE_PATH, {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      })
      if (profileResponse.ok) {
        try {
          const profile = await profileResponse.json()
          syncUserIdCookieFromProfile(profile)
        } catch {
          /* ignore profile parse errors; session still valid */
        }
        return true
      }
      this._clearStaleAuthHints()
      return false
    } catch {
      this._clearStaleAuthHints()
      return false
    }
  }

  _clearStaleAuthHints() {
    if (this._getCookie('userId')) {
      this._clearUserIdCookie()
    }
    if (typeof this._clearAuthToken === 'function') {
      this._clearAuthToken()
    }
  }
}
