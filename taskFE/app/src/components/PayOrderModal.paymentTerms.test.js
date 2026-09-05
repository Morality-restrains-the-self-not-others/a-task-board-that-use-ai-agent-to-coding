/**
 * 回归：PayOrderModal 支付条款门禁 props / 文案约定。
 */
import { describe, it, expect } from 'vitest'
import { PAYMENT_TERMS_CONSENT_REQUIRED, isPaymentTermsConsentRequired } from '../utils/paymentTermsConsent.js'

describe('PayOrderModal payment terms gate', () => {
  it('识别服务端签署门禁码以驱动条款面板', () => {
    expect(isPaymentTermsConsentRequired({ reason_code: PAYMENT_TERMS_CONSENT_REQUIRED })).toBe(true)
  })
})
