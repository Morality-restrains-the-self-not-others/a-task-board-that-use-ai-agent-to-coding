if (!process.env.VITEST) {
  console.log('[skip] useSystemAdminLicenseAgreement.paymentTerms.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach } = await import('vitest')

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))
vi.mock('../utils/paymentTermsConsent.js', () => ({
  fetchCurrentPaymentTerms: vi.fn(),
}))

const { apiFetch } = await import('../utils/apiUtils.js')
const { fetchCurrentPaymentTerms } = await import('../utils/paymentTermsConsent.js')
const { useSystemAdminLicenseAgreement } = await import('./useSystemAdminLicenseAgreement.js')

/**
 * OPT-20260819-026：无生效支付服务条款时，管理端法律文档页须提示「支付服务条款未发布」。
 * 服务端门禁 fail-open（仅 payment_terms_consent_skipped_no_active 结构化日志），
 * 前端依据公开接口 current 返回 null 判断缺口，供运营及时补发。
 */
describe('useSystemAdminLicenseAgreement 支付条款未发布告警', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('无生效支付条款（404→null）时 paymentTermsPublished=false', async () => {
    fetchCurrentPaymentTerms.mockResolvedValue(null)

    const { checkPaymentTermsPublished, paymentTermsPublished } = useSystemAdminLicenseAgreement()
    await checkPaymentTermsPublished()

    expect(paymentTermsPublished.value).toBe(false)
  })

  it('存在生效支付条款时 paymentTermsPublished=true', async () => {
    fetchCurrentPaymentTerms.mockResolvedValue({ id: 'la-pay-1', title: '支付服务条款' })

    const { checkPaymentTermsPublished, paymentTermsPublished } = useSystemAdminLicenseAgreement()
    await checkPaymentTermsPublished()

    expect(paymentTermsPublished.value).toBe(true)
  })

  it('查询接口网络错误时默认视为已发布，避免误报', async () => {
    fetchCurrentPaymentTerms.mockRejectedValue(new Error('network down'))

    const { checkPaymentTermsPublished, paymentTermsPublished } = useSystemAdminLicenseAgreement()
    await checkPaymentTermsPublished()

    expect(paymentTermsPublished.value).toBe(true)
  })

  it('创建生效支付条款后重新检查并清除告警', async () => {
    apiFetch
      .mockResolvedValueOnce({ ok: true, json: async () => ({ id: 'la-new' }) }) // handleCreate POST
      .mockResolvedValueOnce({ ok: true, json: async () => [] }) // refreshAgreements after create
    fetchCurrentPaymentTerms.mockResolvedValue({ id: 'la-new' })

    const { handleCreate, paymentTermsPublished } = useSystemAdminLicenseAgreement()
    await handleCreate()

    expect(paymentTermsPublished.value).toBe(true)
  })
})
}
