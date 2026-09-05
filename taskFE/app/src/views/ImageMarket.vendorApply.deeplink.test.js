// @vitest-environment jsdom
// OPT-20260904-002: 厂商申请展开态可用 URL/query（?vendor_apply=1）深链复现。
if (!process.env.VITEST) {
  console.log('[skip] ImageMarket.vendorApply.deeplink.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  const replaceMock = vi.hoisted(() => vi.fn())
  const routeState = vi.hoisted(() => ({ query: {} }))

  vi.mock('../utils/apiUtils', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      apiFetch: apiFetchMock,
    }
  })
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: '873202574256271360' }, query: routeState.query }),
    useRouter: () => ({ replace: replaceMock }),
  }))

  const { default: ImageMarket } = await import('./ImageMarket.vue')

  beforeEach(() => {
    apiFetchMock.mockReset()
    replaceMock.mockReset()
    routeState.query = { vendor_apply: '1' }
  })

  const jsonResponse = (body, { ok = true, status = 200 } = {}) => ({
    ok,
    status,
    json: async () => body,
    text: async () => JSON.stringify(body),
  })

  const mountImageMarket = async (statusData) => {
    apiFetchMock.mockImplementation(async (url) => {
      if (String(url).includes('vendor-status')) {
        return jsonResponse(statusData)
      }
      if (String(url).includes('phone-status')) {
        return jsonResponse({ has_phone: false })
      }
      return jsonResponse([])
    })
    const wrapper = mount(ImageMarket, {
      global: { stubs: { 'runtime-deps-block': true, ImageAutoRunSteps: true } },
    })
    await flushPromises()
    return wrapper
  }

  const applyForm = (wrapper) => wrapper.find('[data-testid="vendor-app-form"]')
  const applyOpen = (wrapper) => wrapper.find('[data-testid="vendor-apply-open"]')
  const vendorButton = (wrapper) =>
    wrapper.find('a[href^="/api/accounts/sso/ai-provider/vendor/"]')

  describe('ImageMarket.vue ?vendor_apply=1 深链', () => {
    it('none + 邮箱 + 审核开：加载后直接展开申请表（无需点击）', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: true,
        vendor_application_review_enabled: true,
        vendor: null,
      })
      expect(applyOpen(wrapper).exists()).toBe(false)
      expect(applyForm(wrapper).exists()).toBe(true)
      expect(replaceMock).toHaveBeenCalled()
    })

    it('rejected + 邮箱：深链直达重新申请表', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'rejected',
        has_email: true,
        vendor: { id: '1', review_note: '资料不全' },
      })
      expect(applyForm(wrapper).exists()).toBe(true)
      expect(wrapper.text()).toContain('资料不全')
    })

    it('无邮箱态不强制展开：显示绑邮箱引导而非申请表', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: false,
        vendor_application_review_enabled: true,
        vendor: null,
      })
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(false)
      expect(wrapper.text()).toContain('绑定邮箱后申请成为厂商')
    })

    it('qualified + 深链：仍走厂商门户 SSO，不展开申请表', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: true,
        status: 'qualified',
        has_email: true,
        vendor: null,
      })
      expect(vendorButton(wrapper).exists()).toBe(true)
      expect(applyForm(wrapper).exists()).toBe(false)
    })

    it('展开后点取消：表单收起并移除 vendor_apply query', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: true,
        vendor_application_review_enabled: true,
        vendor: null,
      })
      expect(applyForm(wrapper).exists()).toBe(true)
      await wrapper.find('[data-testid="vendor-apply-cancel"]').trigger('click')
      await flushPromises()
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(true)
      const lastQuery = replaceMock.mock.calls.at(-1)[0].query
      expect(lastQuery.vendor_apply).toBeUndefined()
    })
  })
}
