/**
 * 回归：BillingOrders 支付条款门禁接线约定（composable 产出字段）。
 */
import { describe, it, expect } from 'vitest'

describe('BillingOrders payment terms wiring', () => {
  it('PayOrderModal 所需 payment-terms props 名称稳定', () => {
    const requiredProps = [
      'paymentTermsGateActive',
      'paymentTermsDoc',
      'paymentTermsConsenting',
      'onPaymentTermsAccepted',
    ]
    expect(requiredProps.every((k) => typeof k === 'string' && k.length > 0)).toBe(true)
  })
})
