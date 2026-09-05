// @vitest-environment jsdom
/**
 * BillingTransactions：顶部过滤卡过滤器移至表格列头（消耗（元）分类/时间范围/项目名称/成员名称）
 * - 顶部过滤卡不再渲染上述 4 项，仅保留 工作空间名称/任务名称
 * - 表头过滤器变更即自动应用（page=1），请求携带对应后端参数
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactions.columnFilters.test.js requires vitest runtime')
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
      if (u.includes('/billing/units')) {
        return jsonOk([
          { unit_type: 'task', name: '任务', is_active: true },
          { unit_type: 'gitlab_disk', name: 'GitLab 磁盘', is_active: true },
          { unit_type: 'gitlab_traffic', name: 'GitLab 流量费', is_active: true },
        ])
      }
      if (u.includes('/accounts/members/company_members/')) {
        return jsonOk({ members: [{ user_id: 'u1', member_name: '张三' }] })
      }
      if (u.includes('/api/projects/tenant_id/')) {
        return jsonOk([{ id: 'p1', name: '示例项目A' }])
      }
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
          Teleport: true,
        },
      },
    })
  }

  function listFilteredUrls() {
    return hoistedMocks.apiFetchMock.mock.calls
      .map((c) => String(c[0] || ''))
      .filter((u) => u.includes('/billing/transactions/list_filtered/'))
  }

  describe('BillingTransactions 表头过滤器（举一反三）', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      // useBillingTransactions.js 以全局形式调用 apiFetch（非 import）
      global.apiFetch = hoistedMocks.apiFetchMock
    })

    afterEach(() => {
      delete global.apiFetch
      vi.restoreAllMocks()
    })

    it('顶部过滤卡不再渲染 时间范围/消耗（元）分类/项目名称/成员名称，仅保留 工作空间/任务', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const filtersCard = wrapper.find('[data-alias="BillingTransactionsFilters"]')
      expect(filtersCard.exists()).toBe(true)
      for (const removed of ['时间范围', '消耗（元）分类', '项目名称', '成员名称']) {
        expect(filtersCard.text()).not.toContain(removed)
      }
      expect(filtersCard.text()).toContain('工作空间名称')
      expect(filtersCard.text()).toContain('任务名称')
    })

    it('消耗（元）分类 select 渲染在表头，选项来自 /billing/units 接口', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const select = wrapper.get('select[aria-label="消耗（元）分类"]')
      const optionTexts = select.findAll('option').map((o) => o.text())
      expect(optionTexts).toEqual(['全部分类', '任务', 'GitLab 磁盘', 'GitLab 流量费'])
    })

    it('切换表头分类为「任务」：请求携带 billing_unit_type=task 且重置到第 1 页', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const before = listFilteredUrls().length
      await wrapper.get('select[aria-label="消耗（元）分类"]').setValue('task')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls.length).toBe(before + 1)
      expect(urls[urls.length - 1]).toContain('billing_unit_type=task')
      expect(urls[urls.length - 1]).toContain('page=1')
    })

    it('变更开始日期：请求携带 start_date', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const input = wrapper.get('input[aria-label="开始日期"]')
      await input.setValue('2026-08-01')

      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).toContain('start_date=2026-08-01')
    })

    it('变更结束日期：请求携带 end_date', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const input = wrapper.get('input[aria-label="结束日期"]')
      await input.setValue('2026-08-07')

      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).toContain('end_date=2026-08-07')
    })

    it('在「项目」列头聚焦并选中选项：请求携带 project_id 且自动应用', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const input = wrapper.get('input[aria-label="项目名称"]')
      await input.trigger('focus')
      await flushPromises()

      const th = wrapper.findAll('thead th')[6]
      const options = th.findAll('.project-filter .px-3.py-2')
      expect(options.length).toBeGreaterThan(0)
      await options[0].trigger('click')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).toContain('project_id=p1')
      expect(urls[urls.length - 1]).toContain('page=1')
    })

    it('在「成员」列头聚焦并选中选项：请求携带 user_id 且自动应用', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const input = wrapper.get('input[aria-label="成员名称"]')
      await input.trigger('focus')
      await flushPromises()

      const th = wrapper.findAll('thead th')[7]
      const options = th.findAll('.user-filter .px-3.py-2')
      expect(options.length).toBeGreaterThan(0)
      await options[0].trigger('click')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).toContain('user_id=u1')
      expect(urls[urls.length - 1]).toContain('page=1')
    })

    it('重置后表头过滤器清空且重新请求不带过滤参数', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      await wrapper.get('select[aria-label="消耗（元）分类"]').setValue('task')
      await flushPromises()

      // 过滤卡重置按钮
      const filtersCard = wrapper.find('[data-alias="BillingTransactionsFilters"]')
      const resetBtn = filtersCard.findAll('button').find((b) => b.text().includes('重置'))
      await resetBtn.trigger('click')
      await flushPromises()

      const urls = listFilteredUrls()
      expect(urls[urls.length - 1]).not.toContain('billing_unit_type=')
    })
  })
}
