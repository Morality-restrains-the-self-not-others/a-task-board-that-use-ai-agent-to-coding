import { describe, it, expect } from 'vitest'
import {
  isAdminGrantRecharge,
  formatRechargeChannel,
  consentStatusLabel,
  licenseDocumentKindLabel,
  unboundPaymentConsents,
  ORPHAN_CONSENT_HEADING,
  ORPHAN_CONSENT_HINT,
} from './adminUserRechargeDisplay.js'

describe('adminUserRechargeDisplay', () => {
  it('识别 admin_grant 流水（金额 0 / 无渠道）', () => {
    const row = { points_source_type: 'admin_grant', amount_points: 0, channel: '' }
    expect(isAdminGrantRecharge(row)).toBe(true)
    expect(formatRechargeChannel(row)).toBe('系统赠送')
    expect(consentStatusLabel(row)).toBe('系统赠送，无需支付签署')
  })

  it('尊重后端 consent_required=false', () => {
    const row = { consent_required: false, consent_note: '系统赠送，无需支付签署' }
    expect(isAdminGrantRecharge(row)).toBe(true)
    expect(consentStatusLabel(row)).toBe('系统赠送，无需支付签署')
  })

  it('用户支付无签署时仍提示无关联签署记录', () => {
    const row = { points_source_type: 'user_recharge_wechat', channel: 'wechat' }
    expect(isAdminGrantRecharge(row)).toBe(false)
    expect(formatRechargeChannel(row)).toBe('wechat')
    expect(consentStatusLabel(row)).toBe('无关联签署记录')
  })

  it('用户支付已挂载 consent 时标注已关联', () => {
    const row = {
      points_source_type: 'user_recharge_wechat',
      channel: 'wechat',
      consent: { id: 'c1', agreement_version: '0.9' },
    }
    expect(consentStatusLabel(row)).toBe('已关联签署')
  })

  it('协议种类用运营可读名称，支付条款含历史别名', () => {
    expect(licenseDocumentKindLabel('service')).toBe('服务协议')
    expect(licenseDocumentKindLabel('recharge_cents')).toBe('支付服务条款')
    expect(licenseDocumentKindLabel('recharge_points')).toBe('支付服务条款')
    expect(licenseDocumentKindLabel('')).toBe('协议')
  })

  it('未挂到支付的签署排除已展示在流水上的 consent id', () => {
    const recharges = [{ consent: { id: 'c-pay' } }]
    const consents = [
      { id: 'c-pay', document_kind: 'recharge_cents' },
      { id: 'c-signup', document_kind: 'service' },
      { id: 'c-old', document_kind: 'recharge_points' },
    ]
    const unbound = unboundPaymentConsents(recharges, consents)
    expect(unbound.map((c) => c.id)).toEqual(['c-signup', 'c-old'])
  })

  it('区块标题不用「未绑定流水」行话，并说明并非签署缺失', () => {
    expect(ORPHAN_CONSENT_HEADING).toBe('其他协议签署')
    expect(ORPHAN_CONSENT_HEADING).not.toContain('未绑定流水')
    expect(ORPHAN_CONSENT_HINT).toMatch(/不表示签署缺失/)
    expect(ORPHAN_CONSENT_HINT).toMatch(/服务协议/)
  })
})
