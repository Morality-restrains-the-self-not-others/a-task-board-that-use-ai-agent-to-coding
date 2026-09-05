import { fetchRegistrationInvitePolicy } from './registrationInviteUtils.js'

export async function fetchInvitePolicyEnabled(apiFetch) {
  try {
    const policy = await fetchRegistrationInvitePolicy(apiFetch)
    return Boolean(policy.enabled)
  } catch {
    return false
  }
}

export async function fetchCurrentPrivacyPolicy(apiFetch, safeJson) {
  try {
    const r = await apiFetch('/api/privacy-policy/public/current/', { method: 'GET', headers: { Accept: 'application/json' } })
    if (r.ok) {
      return { policy: await safeJson(r, null), error: '' }
    }
    return { policy: null, error: '系统尚未发布隐私条款，无法注册。请联系管理员。' }
  } catch {
    return { policy: null, error: '无法加载隐私条款。' }
  }
}

export async function fetchCurrentLicenseAgreement(apiFetch, safeJson) {
  try {
    const r = await apiFetch('/api/license-agreement/public/current/', { method: 'GET', headers: { Accept: 'application/json' } })
    if (r.ok) {
      return { agreement: await safeJson(r, null), error: '' }
    }
    return { agreement: null, error: '系统尚未发布服务协议，无法注册。请联系管理员。' }
  } catch {
    return { agreement: null, error: '无法加载服务协议。' }
  }
}
