// @vitest-environment jsdom
/**
 * BillingTransactions：交易类型过滤器从顶部过滤卡移至表格「类型」列头
 * - 顶部过滤卡不再渲染「交易类型」select
 * - 表格「类型」列头内嵌交易类型 select（全部类型/入账/消耗/退款）
 * - 切换表头 select 立即生效：更新 filters.transactionType 并重新请求（page=1）
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactions.typeHeaderFilter.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: { tenant: '873472655125147648' },
      path: '/tenant/873472655125147648/billing/transactions/',
      query: {},
    },
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
    parseCompanyMembersResponse: (data) => ({
      members: Array.isArray(data) ? data : (data?.members || []),
      meta: {},
    }),
  }))

  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: () => '',
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => ({ push: vi.fn() }),
  }))

  const BillingTransactions = (await import('./BillingTransactions.vue')).default

  function jsonOk(body) {
    return {
      ok: true,
      status: 200,
      json: async () => body,
    }
  }

  function stubDefaultApis() {
    hoistedMocks.apiFetchMock.mockImplementation(async (url) => {
      const u = String(url || '')
      if (u.includes('/billing/transactions/list_filtered/')) {
        return jsonOk({ results: [], total: 0, page_size: 20 })
      }
      if (u.includes('/billing/units')) return jsonOk([])
      if (u.includes('/accounts/members/company_members/')) return jsonOk({ members: [] })
      return jsonOk([])
    })
  }

  function mountPage() {
    return mount(BillingTransactions, {
      global: {
        stubs: {
          'router-link': {
            props: ['to'],
            template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
          },
        },
      },
    })
  }

  function listFilteredUrls() {
    return hoistedMocks.apiFetchMock.mock.calls
      .map((c) => String(c[0] || ''))
      .filter((u) => u.includes('/billing/transactions/list_filtered/'))
  }

  describe('BillingTransactions 交易类型表头过滤器', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      // useBillingTransactions.js 以全局形式调用 apiFetch（非 import）
      global.apiFetch = hoistedMocks.apiFetchMock
    })

    afterEach(() => {
      delete global.apiFetch
      vi.restoreAllMocks()
    })

    it('顶部过滤卡不再渲染「交易类型」，表格「类型」列头内嵌交易类型 select', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const filtersCard = wrapper.find('[data-alias="BillingTransactionsFilters"]')
      expect(filtersCard.exists()).toBe(true)
      expect(filtersCard.text()).not.toContain('交易类型')

      const typeSelect = wrapper.get('select[aria-label="交易类型"]')
      expect(typeSelect.exists()).toBe(true)
      expect(typeSelect.element.value).toBe('')
    })

    it('切换表头 select 为「消耗」：请求携带 transaction_type=consumption 且重置到第 1 页', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const before = listFilteredUrls().length
      await wrapper.get('select[aria-label="交易类型"]').setValue('consumption')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls.length).toBe(before + 1)
      expect(urls[urls.length - 1]).toContain('transaction_type=consumption')
      expect(urls[urls.length - 1]).toContain('page=1')
    })

    it('切换表头 select 为「入账」：请求携带 transaction_type=recharge', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      await wrapper.get('select[aria-label="交易类型"]').setValue('recharge')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).toContain('transaction_type=recharge')
    })

    it('切换回「全部类型」：请求不再携带 transaction_type 参数', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      await wrapper.get('select[aria-label="交易类型"]').setValue('')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).not.toContain('transaction_type=')
    })
  })
}
