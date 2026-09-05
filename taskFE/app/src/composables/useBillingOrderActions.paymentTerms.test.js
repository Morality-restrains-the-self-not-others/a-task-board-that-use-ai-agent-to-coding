// @vitest-environment jsdom
/**
 * 回归：支付服务条款签署门禁 — 服务端 PAYMENT_TERMS_CONSENT_REQUIRED 时弹条款并重试。
 */
if (!process.env.VITEST) {
  console.log('[skip] useBillingOrderActions.paymentTerms.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))

const { useBillingOrderActions } = await import('./useBillingOrderActions.js')

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return {
    ok,
    status,
    traceId: '',
    json: async () => body,
  }
}

describe('useBillingOrderActions payment terms consent', () => {
  beforeEach(() => {
    hoisted.apiFetch.mockReset()
  })

  it('支付返回 PAYMENT_TERMS_CONSENT_REQUIRED 时打开条款门禁', async () => {
    hoisted.apiFetch.mockImplementation(async (url, opts) => {
      if (String(url).includes('phone-verification-status')) {
        return jsonResponse({ required: false, has_phone: true, sms_verified: true })
      }
      if (String(url).includes('/pay/') && opts?.method === 'POST') {
        return jsonResponse(
          { error: '请先阅读并同意支付服务条款后再支付', reason_code: 'PAYMENT_TERMS_CONSENT_REQUIRED' },
          { ok: false, status: 403 },
        )
      }
      if (String(url).includes('license-agreement/public/current')) {
        return jsonResponse({ id: 'agree-1', title: '支付服务条款', content: '条款正文', version: '0.9' })
      }
      return jsonResponse({})
    })

    const actions = useBillingOrderActions({ tenantId: ref('t1'), onOrderChanged: () => {} })
    await actions.openPayModal({ id: 'ord-1', order_number: 'O1', total_yuan: '0.55' })

    expect(actions.payModalVisible.value).toBe(true)
    expect(actions.paymentTermsGateActive.value).toBe(true)
    expect(actions.paymentTermsDoc.value?.id).toBe('agree-1')
    expect(actions.payCodeUrl.value).toBe('')
  })

  it('同意条款后带 consent_id 重试支付', async () => {
    let payBodies = []
    hoisted.apiFetch.mockImplementation(async (url, opts) => {
      if (String(url).includes('phone-verification-status')) {
        return jsonResponse({ required: false, has_phone: true, sms_verified: true })
      }
      if (String(url).includes('/pay/') && opts?.method === 'POST') {
        const body = JSON.parse(opts.body || '{}')
        payBodies.push(body)
        if (!body.consent_id) {
          return jsonResponse(
            { reason_code: 'PAYMENT_TERMS_CONSENT_REQUIRED', error: 'need consent' },
            { ok: false, status: 403 },
          )
        }
        return jsonResponse({ code_url: 'weixin://ok', out_trade_no: 'OUT1', mode: 'mock' })
      }
      if (String(url).includes('license-agreement/public/current')) {
        return jsonResponse({ id: 'agree-1', title: '支付服务条款', content: '正文', version: '0.9' })
      }
      if (String(url).includes('license-agreement/consent') && opts?.method === 'POST') {
        return jsonResponse({ ok: true, consent_id: 'consent-99' })
      }
      if (String(url).includes('/billing/orders/')) {
        return jsonResponse({ status: 'pending' })
      }
      return jsonResponse({})
    })

    const actions = useBillingOrderActions({ tenantId: ref('t1'), onOrderChanged: () => {} })
    await actions.openPayModal({ id: 'ord-2', order_number: 'O2', total_yuan: '1.00' })
    expect(actions.paymentTermsGateActive.value).toBe(true)

    await actions.onPaymentTermsAccepted()
    expect(actions.paymentTermsGateActive.value).toBe(false)
    expect(actions.payCodeUrl.value).toBe('weixin://ok')
    expect(payBodies.some((b) => b.consent_id === 'consent-99')).toBe(true)
  })
})
}
