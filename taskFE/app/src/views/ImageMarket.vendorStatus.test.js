// @vitest-environment jsdom
// 镜像市场四态：申请 CTA→展开表单 / 审核中 / SSO / 绑邮箱。pending 禁止 SSO href。
if (!process.env.VITEST) {
  console.log('[skip] ImageMarket.vendorStatus.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const apiFetchMock = vi.hoisted(() => vi.fn())
  vi.mock('../utils/apiUtils', async (importOriginal) => {
    const actual = await importOriginal()
    return {
      ...actual,
      apiFetch: apiFetchMock,
    }
  })
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: '873202574256271360' }, query: {} }),
    useRouter: () => ({ replace: vi.fn() }),
  }))

  const { default: ImageMarket } = await import('./ImageMarket.vue')

  beforeEach(() => {
    apiFetchMock.mockReset()
  })

  const jsonResponse = (body, { ok = true, status = 200, traceId = '' } = {}) => ({
    ok,
    status,
    traceId,
    json: async () => body,
    text: async () => JSON.stringify(body),
  })

  const mountImageMarket = async (statusData, extras = {}) => {
    apiFetchMock.mockImplementation(async (url) => {
      if (String(url).includes('vendor-status')) {
        if (extras.statusFail) {
          return jsonResponse({ detail: 'boom' }, { ok: false, status: 502, traceId: 'trace-status-fail' })
        }
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

  const vendorButton = (wrapper) =>
    wrapper.find('a[href^="/api/accounts/sso/ai-provider/vendor/"]')
  const applyForm = (wrapper) => wrapper.find('[data-testid="vendor-app-form"]')
  const applyOpen = (wrapper) => wrapper.find('[data-testid="vendor-apply-open"]')

  describe('ImageMarket.vue 厂商申请四态', () => {
    it('qualified：渲染「厂商门户（SSO）」主按钮，无申请表单', async () => {
      const wrapper = await mountImageMarket({ is_vendor: true, status: 'qualified', has_email: true, vendor: null })
      expect(vendorButton(wrapper).exists()).toBe(true)
      expect(vendorButton(wrapper).text()).toContain('厂商门户')
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(false)
    })

    it('pending：渲染审核中且无 SSO href', async () => {
      const wrapper = await mountImageMarket({ is_vendor: false, status: 'pending', has_email: true, vendor: null })
      expect(vendorButton(wrapper).exists()).toBe(false)
      expect(wrapper.text()).toContain('审核中')
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(false)
    })

    it('pending + 审核关闭：仍无 SSO href', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'pending',
        has_email: true,
        vendor_application_review_enabled: false,
        vendor: null,
      })
      expect(vendorButton(wrapper).exists()).toBe(false)
      expect(wrapper.text()).toContain('审核中')
      expect(applyForm(wrapper).exists()).toBe(false)
    })

    it('none + 邮箱 + 审核开：默认仅申请 CTA，点击后展开表单，无 SSO', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: true,
        vendor_application_review_enabled: true,
        vendor: null,
      })
      expect(vendorButton(wrapper).exists()).toBe(false)
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(true)
      expect(applyOpen(wrapper).text()).toContain('申请成为厂商门户')
      await applyOpen(wrapper).trigger('click')
      await flushPromises()
      expect(applyForm(wrapper).exists()).toBe(true)
      expect(applyOpen(wrapper).exists()).toBe(false)
      expect(wrapper.text()).toContain('申请成为厂商门户')
    })

    it('rejected：默认重新申请 CTA，点击后展开表单，无 SSO', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'rejected',
        has_email: true,
        vendor: { id: '1', review_note: '资料不全' },
      })
      expect(vendorButton(wrapper).exists()).toBe(false)
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(true)
      expect(applyOpen(wrapper).text()).toContain('申请被驳回')
      await applyOpen(wrapper).trigger('click')
      await flushPromises()
      expect(applyForm(wrapper).exists()).toBe(true)
      expect(wrapper.text()).toContain('资料不全')
    })

    it('展开后点取消收起表单并恢复申请 CTA', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: true,
        vendor_application_review_enabled: true,
        vendor: null,
      })
      await applyOpen(wrapper).trigger('click')
      await flushPromises()
      expect(applyForm(wrapper).exists()).toBe(true)
      await wrapper.find('[data-testid="vendor-apply-cancel"]').trigger('click')
      await flushPromises()
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(true)
    })

    it('审核关闭 + 有邮箱：none 态直接显示厂商门户 SSO', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: true,
        vendor_application_review_enabled: false,
        vendor: null,
      })
      expect(vendorButton(wrapper).exists()).toBe(true)
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(false)
    })

    it('审核关闭 + 无邮箱：引导绑定邮箱，默认不展开申请表', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: false,
        vendor_application_review_enabled: false,
        vendor: null,
      })
      expect(vendorButton(wrapper).exists()).toBe(false)
      expect(wrapper.text()).toContain('绑定邮箱后申请成为厂商')
      const bind = wrapper.find('a[href*="sso_error=email_required"]')
      expect(bind.exists()).toBe(true)
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(false)
    })

    it('审核开启 + 无邮箱：绑邮箱 CTA，默认不展开申请表，无 SSO', async () => {
      const wrapper = await mountImageMarket({
        is_vendor: false,
        status: 'none',
        has_email: false,
        vendor_application_review_enabled: true,
        vendor: null,
      })
      expect(vendorButton(wrapper).exists()).toBe(false)
      expect(wrapper.text()).toContain('绑定邮箱后申请成为厂商')
      expect(applyForm(wrapper).exists()).toBe(false)
      expect(applyOpen(wrapper).exists()).toBe(false)
    })

    it('vendor-status 非 2xx：错误节点带 data-traceId', async () => {
      const wrapper = await mountImageMarket({}, { statusFail: true })
      const err = wrapper.find('[data-traceId="trace-status-fail"]')
      expect(err.exists()).toBe(true)
    })
  })
}
