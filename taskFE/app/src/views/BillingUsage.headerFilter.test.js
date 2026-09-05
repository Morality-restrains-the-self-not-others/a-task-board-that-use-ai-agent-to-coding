// @vitest-environment jsdom
/**
 * BillingUsage：过滤条件卡片合并到表格表头对应字段
 * - 顶部不再渲染独立过滤卡片（无「过滤条件」标题卡片）
 * - 表格 thead 内第二行（data-alias="BillingUsageFiltersRow"）承载全部过滤控件，
 *   且每个控件位于其对应表头列的单元格中：
 *     使用时间→开始/结束日期、计费单元→单元类型 select、项目→项目搜索、
 *     用户→成员搜索、工作空间→工作空间搜索、任务→任务搜索、描述→应用过滤/重置
 * - 应用过滤 → 请求携带过滤参数且 page=1；重置 → 参数清空
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingUsage.headerFilter.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: {
      params: { tenant: '873472655125147648' },
      path: '/tenant/873472655125147648/billing/usage/',
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

  const BillingUsage = (await import('./BillingUsage.vue')).default

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
      if (u.includes('/billing/usages/')) {
        return jsonOk({ results: [], total: 0, page_size: 20 })
      }
      if (u.includes('/billing/units')) {
        return jsonOk([
          { id: 'u1', unit_type: 'task_gitlab_disk', name: '任务GitLab磁盘', is_active: true },
        ])
      }
      if (u.includes('/accounts/members/company_members/')) return jsonOk({ members: [] })
      return jsonOk([])
    })
  }

  function mountPage() {
    return mount(BillingUsage, {
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

  function usageUrls() {
    return hoistedMocks.apiFetchMock.mock.calls
      .map((c) => String(c[0] || ''))
      .filter((u) => u.includes('/billing/usages/'))
  }

  describe('BillingUsage 表头过滤合并', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      // useBillingUsage.js 以全局形式调用 apiFetch（非 import）
      global.apiFetch = hoistedMocks.apiFetchMock
    })

    afterEach(() => {
      delete global.apiFetch
      vi.restoreAllMocks()
    })

    it('顶部不再渲染独立过滤卡片，过滤控件位于表格 thead 过滤行内', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      expect(wrapper.find('[data-alias="BillingUsageFilters"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('过滤条件')

      const filterRow = wrapper.get('[data-alias="BillingUsageFiltersRow"]')
      expect(filterRow.element.parentElement.tagName).toBe('THEAD')
      expect(filterRow.element.parentElement.children[1]).toBe(filterRow.element)
    })

    it('过滤控件与表头列一一对应（时间/单元/项目/用户/工作空间/任务/按钮）', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      const headerCells = wrapper.get('thead tr:first-child').findAll('th')
      const filterCells = wrapper.get('[data-alias="BillingUsageFiltersRow"]').findAll('td')
      expect(filterCells).toHaveLength(headerCells.length)

      const colOf = (alias) => filterCells[headerCells.findIndex((th) => th.text().includes(alias))]

      expect(colOf('使用时间').find('[data-alias="BillingUsageStartDate"]').exists()).toBe(true)
      expect(colOf('使用时间').find('[data-alias="BillingUsageEndDate"]').exists()).toBe(true)

      const unitSelect = colOf('计费单元').get('[data-alias="BillingUsageUnitType"]')
      expect(unitSelect.element.value).toBe('')
      expect(unitSelect.findAll('option')[0].text()).toBe('全部类型')

      expect(colOf('项目').find('[data-alias="BillingUsageProjectSearch"]').exists()).toBe(true)
      expect(colOf('用户').find('[data-alias="BillingUsageUserSearch"]').exists()).toBe(true)
      expect(colOf('工作空间').find('[data-alias="BillingUsageWorkspaceSearch"]').exists()).toBe(true)
      expect(colOf('任务').find('[data-alias="BillingUsageTaskSearch"]').exists()).toBe(true)

      expect(colOf('描述').find('[data-alias="BillingUsageApply"]').text()).toBe('应用过滤')
      expect(colOf('描述').find('[data-alias="BillingUsageReset"]').text()).toBe('重置')
    })

    it('应用过滤：请求携带 start_date/end_date/billing_unit_type 且重置到第 1 页', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      await wrapper.get('[data-alias="BillingUsageStartDate"]').setValue('2026-08-01')
      await wrapper.get('[data-alias="BillingUsageEndDate"]').setValue('2026-08-07')
      await wrapper.get('[data-alias="BillingUsageUnitType"]').setValue('task_gitlab_disk')
      await wrapper.get('[data-alias="BillingUsageApply"]').trigger('click')
      await flushPromises()

      const urls = usageUrls()
      expect(urls[urls.length - 1]).toContain('start_date=2026-08-01')
      expect(urls[urls.length - 1]).toContain('end_date=2026-08-07')
      expect(urls[urls.length - 1]).toContain('billing_unit_type=task_gitlab_disk')
      expect(urls[urls.length - 1]).toContain('page=1')
    })

    it('重置：过滤参数清空，请求不带过滤条件', async () => {
      stubDefaultApis()
      const wrapper = mountPage()
      await flushPromises()

      await wrapper.get('[data-alias="BillingUsageStartDate"]').setValue('2026-08-01')
      await wrapper.get('[data-alias="BillingUsageUnitType"]').setValue('task_gitlab_disk')
      await wrapper.get('[data-alias="BillingUsageReset"]').trigger('click')
      await flushPromises()

      const urls = usageUrls()
      expect(urls[urls.length - 1]).not.toContain('start_date=')
      expect(urls[urls.length - 1]).not.toContain('end_date=')
      expect(urls[urls.length - 1]).not.toContain('billing_unit_type=')
      expect(urls[urls.length - 1]).toContain('page=1')
    })
  })
}
