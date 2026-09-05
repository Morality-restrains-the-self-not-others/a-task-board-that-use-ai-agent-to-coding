import { describe, expect, it } from 'vitest'
import {
  canOpenVendorApplyForm,
  showBindEmailCta,
  showPendingCta,
  showVendorApplyOpenCta,
  showVendorSsoButton,
  vendorApplyButtonDisabled,
  vendorApplyButtonText,
} from './imageMarketVendorCta.js'

describe('imageMarketVendorCta', () => {
  it('qualified 显示 SSO', () => {
    expect(showVendorSsoButton({ status: 'qualified', hasEmail: true, reviewEnabled: true })).toBe(true)
  })

  it('审核关闭 + 真实邮箱显示 SSO，不打开表单', () => {
    expect(showVendorSsoButton({ status: 'none', hasEmail: true, reviewEnabled: false })).toBe(true)
    expect(canOpenVendorApplyForm({ status: 'none', hasEmail: true, reviewEnabled: false })).toBe(false)
  })

  it('无邮箱出绑邮箱链 + 申请表，不出 SSO', () => {
    expect(showBindEmailCta({ hasEmail: false })).toBe(true)
    expect(showVendorSsoButton({ status: 'none', hasEmail: false, reviewEnabled: false })).toBe(false)
    expect(canOpenVendorApplyForm({ status: 'none', hasEmail: false, reviewEnabled: true })).toBe(true)
    expect(canOpenVendorApplyForm({ status: 'none', hasEmail: false, reviewEnabled: false })).toBe(true)
  })

  it('none + 邮箱 + 审核开可打开表单', () => {
    expect(canOpenVendorApplyForm({ status: 'none', hasEmail: true, reviewEnabled: true })).toBe(true)
    expect(vendorApplyButtonText('none')).toContain('申请')
    expect(showVendorApplyOpenCta({
      status: 'none', hasEmail: true, reviewEnabled: true, formOpen: false,
    })).toBe(true)
    expect(showVendorApplyOpenCta({
      status: 'none', hasEmail: true, reviewEnabled: true, formOpen: true,
    })).toBe(false)
  })

  it('无邮箱不显示点开申请 CTA（先绑邮箱）', () => {
    expect(showVendorApplyOpenCta({
      status: 'none', hasEmail: false, reviewEnabled: true, formOpen: false,
    })).toBe(false)
  })

  it('pending 审核中且禁用，无表单', () => {
    expect(showPendingCta({ status: 'pending', hasEmail: true })).toBe(true)
    expect(vendorApplyButtonDisabled('pending')).toBe(true)
    expect(canOpenVendorApplyForm({ status: 'pending', hasEmail: true, reviewEnabled: true })).toBe(false)
    expect(showVendorSsoButton({ status: 'pending', hasEmail: true, reviewEnabled: true })).toBe(false)
  })

  it('pending 即使审核关闭也不出 SSO', () => {
    expect(showVendorSsoButton({ status: 'pending', hasEmail: true, reviewEnabled: false })).toBe(false)
    expect(showPendingCta({ status: 'pending', hasEmail: true })).toBe(true)
    expect(canOpenVendorApplyForm({ status: 'pending', hasEmail: true, reviewEnabled: false })).toBe(false)
  })

  it('rejected 可重新申请', () => {
    expect(vendorApplyButtonText('rejected')).toContain('重新申请')
    expect(canOpenVendorApplyForm({ status: 'rejected', hasEmail: true, reviewEnabled: true })).toBe(true)
  })
})
