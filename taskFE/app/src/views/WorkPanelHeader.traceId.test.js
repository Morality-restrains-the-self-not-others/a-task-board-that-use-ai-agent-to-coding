// @vitest-environment jsdom
// header-title-row 的 data-traceId 绑定（集成级：真实 WorkspaceSwitcher 事件链）：
// 工作空间列表加载失败 → WorkspaceSwitcher emit workspace-load-error → 标题行挂载真实请求 traceId；
// 后续加载成功 → emit workspace-loaded → 清除。对齐全站「data-traceId=请求失败 traceId」约定。
if (!process.env.VITEST) {
  console.log('[skip] WorkPanelHeader.traceId.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const { apiFetchMock } = vi.hoisted(() => ({ apiFetchMock: vi.fn() }))
  vi.mock('vue-router', () => ({
    useRouter: () => ({ push: vi.fn() }),
  }))
  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => apiFetchMock(...args),
  }))

  const { default: WorkPanelHeader } = await import('./WorkPanelHeader.vue')

  // Vue 编译器将模板中的 data-traceId 渲染为小写属性 data-traceid（attribute 名大小写不敏感，
  // 与全站既有 134 处 data-traceId 行为一致；getAttribute/CSS 选择器均能命中）。
  // 此处按真实消费方（dom-trace.js / E2E）的大小写不敏感方式读取。
  const titleRowTraceId = (wrapper) => {
    const row = wrapper.get('[data-alias="header-title-row"]').element
    const lowerName = 'data-traceid'
    for (const attr of row.attributes) {
      if (attr.name.toLowerCase() === lowerName) return attr.value
    }
    return undefined
  }

  const summaryRowTraceId = (wrapper) => {
    const row = wrapper.get('[data-alias="header-summary-row"]').element
    const lowerName = 'data-traceid'
    for (const attr of row.attributes) {
      if (attr.name.toLowerCase() === lowerName) return attr.value
    }
    return undefined
  }

  const mountHeader = () =>
    mount(WorkPanelHeader, {
      props: { tenantId: '123', machineSummary: null, machineSummaryLabel: '' },
    })

  describe('WorkPanelHeader 标题行 data-traceId 绑定', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      vi.spyOn(console, 'warn').mockImplementation(() => {})
    })
    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('默认（无加载失败）不挂 data-traceId 属性', async () => {
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => [{ id: 1, name: '用户的工作空间', is_current: true }],
      })
      const wrapper = mountHeader()
      await flushPromises()
      expect(titleRowTraceId(wrapper)).toBeUndefined()
    })

    it('工作空间加载失败时标题行挂载失败请求的 traceId', async () => {
      apiFetchMock.mockResolvedValue({ ok: false, traceId: 'trace-ws-001' })
      const wrapper = mountHeader()
      await flushPromises()
      await flushPromises()
      expect(titleRowTraceId(wrapper)).toBe('trace-ws-001')
    })

    it('失败后重新加载成功则清除 data-traceId', async () => {
      apiFetchMock.mockResolvedValueOnce({ ok: false, traceId: 'trace-ws-002' })
      apiFetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => [{ id: 1, name: '用户的工作空间', is_current: true }],
      })
      const wrapper = mountHeader()
      await flushPromises()
      expect(titleRowTraceId(wrapper)).toBe('trace-ws-002')

      await wrapper.setProps({ refreshTrigger: 1 })
      await flushPromises()
      expect(titleRowTraceId(wrapper)).toBeUndefined()
    })

    it('空 traceId 的失败不挂属性', async () => {
      apiFetchMock.mockResolvedValue({ ok: false, traceId: '' })
      const wrapper = mountHeader()
      await flushPromises()
      await flushPromises()
      expect(titleRowTraceId(wrapper)).toBeUndefined()
    })
  })

  describe('WorkPanelHeader 摘要行 data-traceId 绑定（OPT-20260809-012）', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      // WorkspaceSwitcher 挂载即加载工作空间列表：默认给成功响应，摘要行测试只关注 prop 绑定
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => [{ id: 1, name: '用户的工作空间', is_current: true }],
      })
      vi.spyOn(console, 'warn').mockImplementation(() => {})
    })
    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('默认（无轮询失败）不挂 data-traceId 属性', async () => {
      const wrapper = mountHeader()
      await flushPromises()
      expect(summaryRowTraceId(wrapper)).toBeUndefined()
    })

    it('machineSummaryErrorTraceId 非空时摘要行挂载该 traceId', async () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          tenantId: '123',
          machineSummary: null,
          machineSummaryLabel: '',
          machineSummaryErrorTraceId: 'trace-summary-004',
        },
      })
      await flushPromises()
      expect(summaryRowTraceId(wrapper)).toBe('trace-summary-004')
    })

    it('traceId 清除（空串）后摘要行不再挂属性', async () => {
      const wrapper = mount(WorkPanelHeader, {
        props: {
          tenantId: '123',
          machineSummary: null,
          machineSummaryLabel: '',
          machineSummaryErrorTraceId: 'trace-summary-005',
        },
      })
      await flushPromises()
      expect(summaryRowTraceId(wrapper)).toBe('trace-summary-005')

      await wrapper.setProps({ machineSummaryErrorTraceId: '' })
      await flushPromises()
      expect(summaryRowTraceId(wrapper)).toBeUndefined()
    })
  })

  describe('WorkPanelHeader 摘要陈旧角标（OPT-20260809-029）', () => {
    beforeEach(() => {
      apiFetchMock.mockReset()
      apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => [{ id: 1, name: '用户的工作空间', is_current: true }],
      })
      vi.spyOn(console, 'warn').mockImplementation(() => {})
    })
    afterEach(() => {
      vi.restoreAllMocks()
    })

    const mountWithStale = (staleSince) =>
      mount(WorkPanelHeader, {
        props: {
          tenantId: '123',
          machineSummary: null,
          machineSummaryLabel: '',
          machineSummaryStaleSince: staleSince,
        },
      })

    it('默认（staleSince 为 null）不渲染陈旧角标', async () => {
      const wrapper = mountWithStale(null)
      await flushPromises()
      expect(wrapper.find('[data-alias="machine-summary-stale-badge"]').exists()).toBe(false)
    })

    it('staleSince 有值（number）时渲染灰色角标 + 「数据可能已过期」', async () => {
      const wrapper = mountWithStale(1723200000000)
      await flushPromises()
      const badge = wrapper.get('[data-alias="machine-summary-stale-badge"]')
      expect(badge.text()).toContain('数据可能已过期')
    })

    it('恢复（staleSince 置回 null）后角标消失', async () => {
      const wrapper = mountWithStale(1723200000000)
      await flushPromises()
      expect(wrapper.find('[data-alias="machine-summary-stale-badge"]').exists()).toBe(true)

      await wrapper.setProps({ machineSummaryStaleSince: null })
      await flushPromises()
      expect(wrapper.find('[data-alias="machine-summary-stale-badge"]').exists()).toBe(false)
    })
  })
}
