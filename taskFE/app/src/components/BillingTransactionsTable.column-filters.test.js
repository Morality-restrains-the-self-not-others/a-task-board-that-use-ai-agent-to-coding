// @vitest-environment jsdom
/**
 * BillingTransactionsTable：顶部过滤卡过滤器移至表格列头（举一反三，延续 67d4a7d 交易类型列头模式）
 * - 消耗（元）分类 → 第 3 列「消耗（元）分类」列头 select（全部分类/任务/GitLab 磁盘/GitLab 流量费）
 * - 时间范围 → 第 1 列「交易时间」列头开始/结束日期输入
 * - 项目名称 → 「项目」列头搜索输入 + 下拉（Teleport 到 body，fixed 定位避免 overflow-x-auto 裁剪）
 * - 成员名称 → 「成员」列头搜索输入 + 下拉
 * - 所有表头过滤器变更后立即发出 update:filter + apply（自动应用）
 */
if (!process.env.VITEST) {
  console.log('[skip] BillingTransactionsTable.column-filters.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi, afterEach } = await import('vitest')

  afterEach(() => {
    vi.useRealTimers()
  })

  const { default: BillingTransactionsTable } = await import('./BillingTransactionsTable.vue')

  const baseProps = {
    tenantId: 't1',
    transactions: [],
    totalCount: 0,
    currentPage: 1,
    totalPages: 1,
    formatDate: (d) => d,
    getTransactionTypeText: (t) => t,
    paymentSource: '',
    paymentSourceOptions: [],
    transactionTypeFilter: '',
  }

  const billingUnitTypeOptions = [
    { value: 'task', label: '任务' },
    { value: 'gitlab_disk', label: 'GitLab 磁盘' },
    { value: 'gitlab_traffic', label: 'GitLab 流量费' },
  ]

  const projectOptions = [
    { id: 'p1', name: '示例项目A' },
    { id: 'p2', name: '示例项目B' },
  ]
  const userOptions = [
    { id: 'u1', name: '张三' },
    { id: 'u2', name: '李四' },
  ]

  const mountTable = (props = {}) =>
    mount(BillingTransactionsTable, {
      props: { ...baseProps, ...props },
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          // 单测中将 Teleport 原地渲染，便于在 wrapper 内断言下拉内容
          Teleport: true,
        },
      },
    })

  describe('消耗（元）分类列头过滤器', () => {
    it('分类 select 渲染在第 3 列「消耗（元）分类」列头中，选项来自 billingUnitTypeOptions', () => {
      const wrapper = mountTable({ billingUnitTypeOptions })
      const th = wrapper.findAll('thead th')[2]
      const select = th.find('select[aria-label="消耗（元）分类"]')
      expect(select.exists()).toBe(true)
      const optionTexts = select.findAll('option').map((o) => o.text())
      expect(optionTexts).toEqual(['全部分类', '任务', 'GitLab 磁盘', 'GitLab 流量费'])
    })

    it('回显当前选中的分类值', () => {
      const wrapper = mountTable({ billingUnitTypeOptions, billingUnitTypeFilter: 'gitlab_disk' })
      const select = wrapper.find('select[aria-label="消耗（元）分类"]')
      expect(select.element.value).toBe('gitlab_disk')
    })

    it('切换分类时发出 update:filter(billingUnitType, value) 与 apply', async () => {
      const wrapper = mountTable({ billingUnitTypeOptions })
      await wrapper.find('select[aria-label="消耗（元）分类"]').setValue('task')
      expect(wrapper.emitted('update:filter')).toContainEqual(['billingUnitType', 'task'])
      expect(wrapper.emitted('apply')).toHaveLength(1)
    })

    it('切回「全部分类」发出空值', async () => {
      const wrapper = mountTable({ billingUnitTypeOptions, billingUnitTypeFilter: 'task' })
      await wrapper.find('select[aria-label="消耗（元）分类"]').setValue('')
      expect(wrapper.emitted('update:filter')).toContainEqual(['billingUnitType', ''])
    })
  })

  describe('交易时间列头日期过滤器', () => {
    it('「交易时间」列头渲染开始/结束日期输入，回显当前值', () => {
      const wrapper = mountTable({ startDate: '2026-08-01', endDate: '2026-08-07' })
      const th = wrapper.findAll('thead th')[0]
      expect(th.text()).toContain('交易时间')
      const startInput = th.find('input[aria-label="开始日期"]')
      const endInput = th.find('input[aria-label="结束日期"]')
      expect(startInput.exists()).toBe(true)
      expect(endInput.exists()).toBe(true)
      expect(startInput.element.value).toBe('2026-08-01')
      expect(endInput.element.value).toBe('2026-08-07')
    })

    it('变更开始日期发出 update:filter(startDate, value) 与 apply', async () => {
      const wrapper = mountTable()
      const input = wrapper.find('input[aria-label="开始日期"]')
      await input.setValue('2026-08-01')

      expect(wrapper.emitted('update:filter')).toContainEqual(['startDate', '2026-08-01'])
      expect(wrapper.emitted('apply')).toHaveLength(1)
    })

    it('变更结束日期发出 update:filter(endDate, value) 与 apply', async () => {
      const wrapper = mountTable()
      const input = wrapper.find('input[aria-label="结束日期"]')
      await input.setValue('2026-08-07')

      expect(wrapper.emitted('update:filter')).toContainEqual(['endDate', '2026-08-07'])
      expect(wrapper.emitted('apply')).toHaveLength(1)
    })
  })

  describe('项目列头搜索过滤器', () => {
    it('「项目」列头渲染搜索输入，输入时发出 update:projectSearch，300ms 防抖后触发一次 search-projects', async () => {
      vi.useFakeTimers()
      const wrapper = mountTable()
      const th = wrapper.findAll('thead th')[6]
      expect(th.text()).toContain('项目')
      const input = th.find('input[aria-label="项目名称"]')
      expect(input.exists()).toBe(true)
      await input.setValue('示例')
      expect(wrapper.emitted('update:projectSearch')).toEqual([['示例']])
      // 防抖（OPT-20260807-045）：输入后立即不触发，300ms 后触发一次
      expect(wrapper.emitted('search-projects')).toBeUndefined()
      await vi.advanceTimersByTimeAsync(300)
      expect(wrapper.emitted('search-projects')).toHaveLength(1)
    })

    it('聚焦时发出 focus-project', async () => {
      const wrapper = mountTable()
      await wrapper.find('input[aria-label="项目名称"]').trigger('focus')
      expect(wrapper.emitted('focus-project')).toHaveLength(1)
    })

    it('showProjectDropdown 且存在选项时渲染下拉，点击选项发出 select-project', async () => {
      const wrapper = mountTable({ projectSearch: '', projectOptions, showProjectDropdown: true })
      const th = wrapper.findAll('thead th')[6]
      const options = th.findAll('.project-filter .px-3.py-2')
      expect(options).toHaveLength(2)
      await th.findAll('.project-filter .px-3.py-2')[0].trigger('click')
      expect(wrapper.emitted('select-project')).toEqual([[{ id: 'p1', name: '示例项目A' }]])
    })

    it('下拉隐藏时（showProjectDropdown=false）不渲染选项', () => {
      const wrapper = mountTable({ projectOptions })
      const th = wrapper.findAll('thead th')[6]
      expect(th.findAll('.project-filter .px-3.py-2')).toHaveLength(0)
    })
  })

  describe('成员列头搜索过滤器', () => {
    it('「成员」列头渲染搜索输入，输入时发出 update:userSearch，300ms 防抖后触发一次 search-users', async () => {
      vi.useFakeTimers()
      const wrapper = mountTable()
      const th = wrapper.findAll('thead th')[7]
      expect(th.text()).toContain('成员')
      const input = th.find('input[aria-label="成员名称"]')
      expect(input.exists()).toBe(true)
      await input.setValue('张')
      expect(wrapper.emitted('update:userSearch')).toEqual([['张']])
      // 防抖（OPT-20260807-045）：输入后立即不触发，300ms 后触发一次
      expect(wrapper.emitted('search-users')).toBeUndefined()
      await vi.advanceTimersByTimeAsync(300)
      expect(wrapper.emitted('search-users')).toHaveLength(1)
    })

    it('聚焦时发出 focus-user', async () => {
      const wrapper = mountTable()
      await wrapper.find('input[aria-label="成员名称"]').trigger('focus')
      expect(wrapper.emitted('focus-user')).toHaveLength(1)
    })

    it('showUserDropdown 且存在选项时渲染下拉，点击选项发出 select-user', async () => {
      const wrapper = mountTable({ userSearch: '', userOptions, showUserDropdown: true })
      const th = wrapper.findAll('thead th')[7]
      const options = th.findAll('.user-filter .px-3.py-2')
      expect(options).toHaveLength(2)
      await options[1].trigger('click')
      expect(wrapper.emitted('select-user')).toEqual([[{ id: 'u2', name: '李四' }]])
    })
  })

  describe('表头 select 总量', () => {
    it('表格内共 3 处 select：标题栏支付来源 + 交易类型列头 + 消耗（元）分类列头', () => {
      const wrapper = mountTable({ billingUnitTypeOptions })
      expect(wrapper.findAll('select')).toHaveLength(3)
    })
  })
}
