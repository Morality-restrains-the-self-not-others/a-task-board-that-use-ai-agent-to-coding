// @vitest-environment jsdom
/**
 * OPT-20260807-046/048 共用表头过滤器组件回归测试。
 * - HeaderDateFilter：受控日期范围（update:startDate/update:endDate + change 聚合 commit）
 * - HeaderSelectFilter：受控下拉（update:modelValue）
 * - HeaderSearchFilter：受控搜索 + 300ms 防抖 search + focus + select + Teleport fixed 定位
 * - 三组件被 BillingUsageFilters / BillingTransactionsTable / BillingTransactionsFilters 复用
 *   （消除复制粘贴），本测试锁定共用组件契约，防消费方再引入重复实现。
 */
if (!process.env.VITEST) {
  console.log('[skip] HeaderFilters.shared-components.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi, afterEach } = await import('vitest')

  const HeaderDateFilter = (await import('./HeaderDateFilter.vue')).default
  const HeaderSelectFilter = (await import('./HeaderSelectFilter.vue')).default
  const HeaderSearchFilter = (await import('./HeaderSearchFilter.vue')).default

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('HeaderDateFilter', () => {
    it('渲染开始/结束日期输入并回显值', () => {
      const wrapper = mount(HeaderDateFilter, {
        props: { startDate: '2026-08-01', endDate: '2026-08-07' },
      })
      const inputs = wrapper.findAll('input[type="date"]')
      expect(inputs).toHaveLength(2)
      expect(inputs[0].element.value).toBe('2026-08-01')
      expect(inputs[1].element.value).toBe('2026-08-07')
    })

    it('data-alias 透传到对应输入框', () => {
      const wrapper = mount(HeaderDateFilter, {
        props: { startAlias: 'MyStart', endAlias: 'MyEnd' },
      })
      expect(wrapper.get('[data-alias="MyStart"]').attributes('type')).toBe('date')
      expect(wrapper.get('[data-alias="MyEnd"]').attributes('type')).toBe('date')
    })

    it('输入时发出 update:startDate / update:endDate，变更时聚合 commit({key, value})', async () => {
      const wrapper = mount(HeaderDateFilter)
      await wrapper.findAll('input')[0].setValue('2026-08-02')
      expect(wrapper.emitted('update:startDate')).toEqual([['2026-08-02']])
      // setValue 同时触发 input+change → commit 携带 startDate
      expect(wrapper.emitted('commit')).toContainEqual([{ key: 'startDate', value: '2026-08-02' }])

      await wrapper.findAll('input')[1].setValue('2026-08-09')
      expect(wrapper.emitted('update:endDate')).toEqual([['2026-08-09']])
      expect(wrapper.emitted('commit')).toContainEqual([{ key: 'endDate', value: '2026-08-09' }])
    })
  })

  describe('HeaderSelectFilter', () => {
    it('渲染占位 option 与选项列表，回显当前值', () => {
      const options = [
        { value: 'a', label: 'A' },
        { value: 'b', label: 'B' },
      ]
      const wrapper = mount(HeaderSelectFilter, {
        props: { modelValue: 'b', options, placeholder: '全部', ariaLabel: '测试分类' },
      })
      const select = wrapper.get('select[aria-label="测试分类"]')
      expect(select.findAll('option').map((o) => o.text())).toEqual(['全部', 'A', 'B'])
      expect(select.element.value).toBe('b')
    })

    it('切换时发出 update:modelValue(value)', async () => {
      const wrapper = mount(HeaderSelectFilter, {
        props: {
          options: [{ value: 'a', label: 'A' }],
          placeholder: '全部',
          dataAlias: 'TestSelect',
        },
      })
      await wrapper.get('[data-alias="TestSelect"]').setValue('a')
      expect(wrapper.emitted('update:modelValue')).toEqual([['a']])
    })
  })

  describe('HeaderSearchFilter', () => {
    const baseProps = {
      search: '',
      options: [
        { id: 'p1', name: '项目A' },
        { id: 'p2', name: '项目B' },
      ],
      showDropdown: false,
      dataAlias: 'TestSearch',
      ariaLabel: '项目名称',
    }

    it('输入即时发出 update:search，300ms 防抖后发出一次 search', async () => {
      vi.useFakeTimers()
      const wrapper = mount(HeaderSearchFilter, { props: baseProps })
      await wrapper.get('[data-alias="TestSearch"]').setValue('示例')
      expect(wrapper.emitted('update:search')).toEqual([['示例']])
      expect(wrapper.emitted('search')).toBeUndefined()
      await vi.advanceTimersByTimeAsync(300)
      expect(wrapper.emitted('search')).toHaveLength(1)
    })

    it('聚焦时发出 focus，点击选项发出 select(option)', async () => {
      const wrapper = mount(HeaderSearchFilter, {
        props: { ...baseProps, showDropdown: true },
        global: { stubs: { Teleport: true } },
      })
      await wrapper.get('[data-alias="TestSearch"]').trigger('focus')
      expect(wrapper.emitted('focus')).toHaveLength(1)

      const options = wrapper.findAll('.px-3.py-2')
      expect(options).toHaveLength(2)
      await options[1].trigger('click')
      expect(wrapper.emitted('select')).toEqual([[{ id: 'p2', name: '项目B' }]])
    })

    it('下拉隐藏或选项为空时不渲染', () => {
      const wrapper = mount(HeaderSearchFilter, { props: baseProps })
      expect(wrapper.findAll('.px-3.py-2')).toHaveLength(0)
      const emptyOpts = mount(HeaderSearchFilter, {
        props: { ...baseProps, showDropdown: true, options: [] },
      })
      expect(emptyOpts.findAll('.px-3.py-2')).toHaveLength(0)
    })

    it('selectedName 非空时渲染「已选择」行', () => {
      const wrapper = mount(HeaderSearchFilter, {
        props: { ...baseProps, selectedName: '项目A' },
      })
      expect(wrapper.text()).toContain('已选择: 项目A')
    })
  })

  describe('三页共用组件复用（OPT-20260807-046）', () => {
    it('BillingUsageFilters 不再内联 select/搜索下拉，而由共用组件渲染 data-alias', async () => {
      const { default: BillingUsageFilters } = await import('./BillingUsageFilters.vue')
      const wrapper = mount(BillingUsageFilters, {
        props: {
          filters: {},
          billingUnitTypeOptions: [{ value: 'a', label: 'A' }],
        },
      })
      // 时间 → HeaderDateFilter 透传的 data-alias
      expect(wrapper.get('[data-alias="BillingUsageStartDate"]').exists()).toBe(true)
      expect(wrapper.get('[data-alias="BillingUsageEndDate"]').exists()).toBe(true)
      // 单元 select → HeaderSelectFilter 透传
      expect(wrapper.get('[data-alias="BillingUsageUnitType"]').exists()).toBe(true)
      // 四个搜索 → HeaderSearchFilter 透传
      for (const alias of ['BillingUsageProjectSearch', 'BillingUsageUserSearch', 'BillingUsageWorkspaceSearch', 'BillingUsageTaskSearch']) {
        expect(wrapper.get(`[data-alias="${alias}"]`).exists()).toBe(true)
      }
    })

    it('BillingTransactionsTable 交易类型/分类列头 select 由 HeaderSelectFilter 提供（含固定枚举）', async () => {
      const { default: BillingTransactionsTable } = await import('./BillingTransactionsTable.vue')
      const wrapper = mount(BillingTransactionsTable, {
        props: {
          tenantId: 't1',
          transactions: [],
          totalCount: 0,
          currentPage: 1,
          totalPages: 1,
          formatDate: (d) => d,
          getTransactionTypeText: (t) => t,
        },
        global: { stubs: { RouterLink: { template: '<a><slot /></a>' }, Teleport: true } },
      })
      const th = wrapper.findAll('thead th')
      // 交易类型列头 select（固定枚举 入账/消耗/退款）
      const typeSelect = th[1].get('select[aria-label="交易类型"]')
      expect(typeSelect.findAll('option').map((o) => o.text())).toEqual(['全部类型', '入账', '消耗', '退款'])
      // 消耗分类列头 select
      expect(th[2].get('select[aria-label="消耗（元）分类"]').exists()).toBe(true)
    })
  })
}
