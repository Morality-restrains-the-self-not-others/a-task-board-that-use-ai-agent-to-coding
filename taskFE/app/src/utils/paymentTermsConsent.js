import { apiFetch } from '../utils/apiUtils.js'

export const PAYMENT_TERMS_KIND = 'recharge_cents'
export const PAYMENT_TERMS_CONSENT_REQUIRED = 'PAYMENT_TERMS_CONSENT_REQUIRED'

/**
 * 拉取当前生效的支付服务条款；无生效文档时返回 null（服务端门禁也会放行）。
 */
export async function fetchCurrentPaymentTerms() {
  const r = await apiFetch(`/api/license-agreement/public/current/?kind=${PAYMENT_TERMS_KIND}`, {
    method: 'GET',
    credentials: 'include',
    headers: { Accept: 'application/json' },
  })
  if (r.status === 404) return null
  if (!r.ok) {
    const d = await r.json().catch(() => ({}))
    const err = new Error(d.error || '加载支付服务条款失败')
    err.traceId = d.trace_id || d._traceId || r.traceId || ''
    throw err
  }
  return r.json()
}

/**
 * 记录用户对支付服务条款的同意，返回 consent_id。
 */
export async function submitPaymentTermsConsent(agreementId) {
  const r = await apiFetch('/api/license-agreement/consent/', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify({
      license_agreement_id: String(agreementId),
      context: 'order_pay',
    }),
  })
  const d = await r.json().catch(() => ({}))
  if (!r.ok) {
    const err = new Error(d.error || '签署支付服务条款失败')
    err.traceId = d.trace_id || d._traceId || r.traceId || ''
    throw err
  }
  return String(d.consent_id || '')
}

export function isPaymentTermsConsentRequired(payload) {
  return payload?.reason_code === PAYMENT_TERMS_CONSENT_REQUIRED
}
