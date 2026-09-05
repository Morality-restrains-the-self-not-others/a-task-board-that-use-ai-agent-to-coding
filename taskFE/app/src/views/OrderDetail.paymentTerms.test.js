/**
 * 回归：OrderDetail 支付路径须识别 PAYMENT_TERMS_CONSENT_REQUIRED。
 */
import { describe, it, expect } from 'vitest'
import { isPaymentTermsConsentRequired, PAYMENT_TERMS_CONSENT_REQUIRED } from '../utils/paymentTermsConsent.js'

describe('OrderDetail payment terms', () => {
  it('支付失败 reason_code 触发条款门禁', () => {
    expect(isPaymentTermsConsentRequired({ reason_code: PAYMENT_TERMS_CONSENT_REQUIRED })).toBe(true)
  })
})
