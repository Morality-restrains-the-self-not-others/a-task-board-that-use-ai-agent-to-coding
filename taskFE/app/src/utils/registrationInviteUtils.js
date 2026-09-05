/* @alias:util-registration-invite */
const STORAGE_KEY = 'registration_invite_code'
export const REGISTRATION_INVITE_POLICY_PUBLIC_API = '/api/public/registration-invite-policy/'

export function readStoredInviteCode() {
  try {
    const v = sessionStorage.getItem(STORAGE_KEY)
    return typeof v === 'string' ? v.trim().toUpperCase() : ''
  } catch {
    return ''
  }
}

export function writeStoredInviteCode(code) {
  const normalized = String(code || '').trim().toUpperCase()
  try {
    if (normalized) {
      sessionStorage.setItem(STORAGE_KEY, normalized)
    } else {
      sessionStorage.removeItem(STORAGE_KEY)
    }
  } catch {
    /* ignore quota / private mode */
  }
  return normalized
}

export async function fetchRegistrationInvitePolicy(apiFetch) {
  const response = await apiFetch(REGISTRATION_INVITE_POLICY_PUBLIC_API, {
    method: 'GET',
    headers: { Accept: 'application/json' },
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const err = new Error(data?.error || data?.detail || `policy ${response.status}`)
    err.status = response.status
    err.traceId = response.headers?.get?.('X-Trace-Id') || data?.trace_id || data?.traceId || ''
    err.payload = data
    throw err
  }
  return {
    enabled: Boolean(data?.enabled),
    daily_quota: Number(data?.daily_quota) || 0,
    remaining_today: Number(data?.remaining_today) || 0,
  }
}
