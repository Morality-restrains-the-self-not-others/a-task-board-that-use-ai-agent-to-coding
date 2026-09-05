import { describe, it, expect } from 'vitest'
import {
  isPaymentTermsConsentRequired,
  PAYMENT_TERMS_CONSENT_REQUIRED,
} from './paymentTermsConsent.js'

describe('paymentTermsConsent', () => {
  it('识别服务端签署门禁 reason_code', () => {
    expect(isPaymentTermsConsentRequired({ reason_code: PAYMENT_TERMS_CONSENT_REQUIRED })).toBe(true)
    expect(isPaymentTermsConsentRequired({ reason_code: 'KYC_TIER_BLOCKED' })).toBe(false)
    expect(isPaymentTermsConsentRequired({})).toBe(false)
  })
})
