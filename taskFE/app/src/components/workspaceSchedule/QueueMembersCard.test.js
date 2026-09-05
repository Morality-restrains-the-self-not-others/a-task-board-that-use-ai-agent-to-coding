// @vitest-environment jsdom
// 排队任务卡：搜索加入、未启用阻断、离开确认、刷新、错误 data-traceId。
if (!process.env.VITEST) {
  console.log('[skip] QueueMembersCard.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    alert: vi.fn(async () => {}),
    confirm: vi.fn(async () => true),
  }))

  vi.mock('../../utils/modalService.js', () => ({
    default: {
      alert: (...a) => hoisted.alert(...a),
      confirm: (...a) => hoisted.confirm(...a),
    },
  }))

  vi.mock('../../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-queue-card' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  const { default: QueueMembersCard } = await import('./QueueMembersCard.vue')

  function jsonResponse(body, { ok = true, status = 200, traceId = null } = {}) {
    return {
      ok,
      status,
      headers: { get: (k) => (String(k).toLowerCase() === 'x-trace-id' ? traceId : null) },
      json: async () => body,
    }
  }

  const MEMBERS = [
    {
      task_id: 'ta',
      title: '顶层任务',
      depth: 0,
      status: 'queued',
      enqueued_at: '2026-08-24T00:00:00Z',
      href: '/tenant/t1/workspace/ws9/task-detail/ta/',
    },
  ]

  function mountCard(overrides = {}) {
    return mount(QueueMembersCard, {
      props: {
        tenantId: 't1',
        workspaceId: 'ws9',
        snapshot: { schedule_rhythm: { enabled: true } },
        members: MEMBERS,
        loading: false,
        actionError: '',
        actionErrorTraceId: '',
        memberStatusText: (m) => (m.status === 'queued' ? '排队中' : '已入队'),
        formatTime: () => 'formatted',
        patchQueuedAutoRun: vi.fn(async () => true),
        ...overrides,
      },
    })
  }

  describe('QueueMembersCard', () => {
    beforeEach(() => {
      hoisted.alert.mockReset()
      hoisted.confirm.mockReset()
      hoisted.confirm.mockResolvedValue(true)
      window.apiFetch = vi.fn()
    })
    afterEach(() => {
      vi.unstubAllGlobals()
      vi.useRealTimers()
    })

    it('未启用自动调度：加入不调用 PATCH，改为 alert 引导调度设置', async () => {
      const patchQueuedAutoRun = vi.fn(async () => true)
      window.apiFetch = vi.fn().mockResolvedValue(jsonResponse({
        results: [{ id: 'tb', title: '新任务', workspace_id: 'ws9' }],
      }))
      const wrapper = mountCard({
        snapshot: { schedule_rhythm: { enabled: false } },
        patchQueuedAutoRun,
      })
      await wrapper.get('[data-testid="queue-join-toggle"]').trigger('click')
      await wrapper.get('[data-testid="queue-join-search-input"]').setValue('新任务')
      await wrapper.get('[data-testid="queue-join-search-submit"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="queue-join-hit-tb"]').trigger('click')
      await flushPromises()
      expect(hoisted.alert).toHaveBeenCalled()
      expect(String(hoisted.alert.mock.calls[0][0])).toContain('调度设置')
      expect(patchQueuedAutoRun).not.toHaveBeenCalled()
      wrapper.unmount()
    })

    it('已启用：搜索排除已入队后 PATCH true 并传 Idempotency-Key', async () => {
      const patchQueuedAutoRun = vi.fn(async () => true)
      window.apiFetch = vi.fn().mockResolvedValue(jsonResponse({
        results: [
          { id: 'ta', title: '已在队', workspace_id: 'ws9' },
          { id: 'tb', title: '新任务', workspace_id: 'ws9' },
          { id: 'tc', title: '其他空间', workspace_id: 'ws-other' },
        ],
      }))
      const wrapper = mountCard({ patchQueuedAutoRun })
      await wrapper.get('[data-testid="queue-join-toggle"]').trigger('click')
      await wrapper.get('[data-testid="queue-join-search-input"]').setValue('任务')
      await wrapper.get('[data-testid="queue-join-search-submit"]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid="queue-join-hit-ta"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="queue-join-hit-tc"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="queue-join-hit-tb"]').exists()).toBe(true)
      const searchUrl = window.apiFetch.mock.calls[0][0]
      expect(searchUrl).toContain('/api/tasks/search/tenant_id/t1/')
      expect(searchUrl).toContain('workspace_id=ws9')
      await wrapper.get('[data-testid="queue-join-hit-tb"]').trigger('click')
      await flushPromises()
      expect(patchQueuedAutoRun).toHaveBeenCalledWith('tb', true, 'ik-queue-card')
      wrapper.unmount()
    })

    it('连续两次不同关键字点击搜索均发出 GET（OPT-20260827-044）', async () => {
      window.apiFetch = vi.fn().mockResolvedValue(jsonResponse({ results: [] }))
      const wrapper = mountCard()
      await wrapper.get('[data-testid="queue-join-toggle"]').trigger('click')

      await wrapper.get('[data-testid="queue-join-search-input"]').setValue('第一关键字')
      await wrapper.get('[data-testid="queue-join-search-submit"]').trigger('click')
      await flushPromises()
      await wrapper.get('[data-testid="queue-join-search-input"]').setValue('第二关键字')
      await wrapper.get('[data-testid="queue-join-search-submit"]').trigger('click')
      await flushPromises()

      expect(window.apiFetch).toHaveBeenCalledTimes(2)
      const url0 = window.apiFetch.mock.calls[0][0]
      const url1 = window.apiFetch.mock.calls[1][0]
      expect(url0).toContain('/api/tasks/search/tenant_id/t1/')
      expect(url0).not.toBe(url1)
      expect(url0).toContain(encodeURIComponent('第一关键字'))
      expect(url1).toContain(encodeURIComponent('第二关键字'))
      wrapper.unmount()
    })

    it('点搜索按钮取消 pending debounce，不重复发 GET（OPT-20260827-044）', async () => {
      vi.useFakeTimers()
      window.apiFetch = vi.fn().mockResolvedValue(jsonResponse({ results: [] }))
      const wrapper = mountCard()
      await wrapper.get('[data-testid="queue-join-toggle"]').trigger('click')
      await wrapper.get('[data-testid="queue-join-search-input"]').setValue('唯一关键字')
      // watch 已排程 250ms 自动搜索；按钮点击应取消它并立即只发一次 GET
      await wrapper.get('[data-testid="queue-join-search-submit"]').trigger('click')
      await flushPromises()
      expect(window.apiFetch).toHaveBeenCalledTimes(1)
      await vi.advanceTimersByTimeAsync(400)
      await flushPromises()
      expect(window.apiFetch).toHaveBeenCalledTimes(1)
      wrapper.unmount()
    })

    it('离开确认后 PATCH false；取消不发', async () => {
      const patchQueuedAutoRun = vi.fn(async () => true)
      const wrapper = mountCard({ patchQueuedAutoRun })
      await wrapper.get('[data-testid="queue-leave-ta"]').trigger('click')
      await flushPromises()
      expect(hoisted.confirm).toHaveBeenCalled()
      expect(patchQueuedAutoRun).toHaveBeenCalledWith('ta', false, 'ik-queue-card')

      patchQueuedAutoRun.mockClear()
      hoisted.confirm.mockRejectedValueOnce(new Error('cancel'))
      await wrapper.get('[data-testid="queue-leave-ta"]').trigger('click')
      await flushPromises()
      expect(patchQueuedAutoRun).not.toHaveBeenCalled()
      wrapper.unmount()
    })

    it('搜索失败 → 错误节点 data-traceId', async () => {
      window.apiFetch = vi.fn().mockResolvedValue(jsonResponse(
        { error: 'search boom', trace_id: 'tr-search' },
        { ok: false, status: 500, traceId: 'tr-search' },
      ))
      const wrapper = mountCard()
      await wrapper.get('[data-testid="queue-join-toggle"]').trigger('click')
      await wrapper.get('[data-testid="queue-join-search-input"]').setValue('x')
      await wrapper.get('[data-testid="queue-join-search-submit"]').trigger('click')
      await flushPromises()
      const errEl = wrapper.get('[data-testid="queue-join-search-error"]')
      let found = ''
      for (const attr of errEl.element.attributes) {
        if (attr.name.toLowerCase() === 'data-traceid') found = attr.value
      }
      expect(found).toBe('tr-search')
      wrapper.unmount()
    })

    it('刷新发出 refresh 事件', async () => {
      const wrapper = mountCard()
      await wrapper.get('[data-testid="refresh-members"]').trigger('click')
      await flushPromises()
      expect(wrapper.emitted('refresh')).toBeTruthy()
      wrapper.unmount()
    })
  })
}
