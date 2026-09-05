// @vitest-environment jsdom
// WorkspaceQueueSchedule 页面 smoke：工作空间解析（URL 优先）、快照加载渲染、
// 保存调用、错误块挂 data-traceId、成员状态文案、排队任务标题真实 a[href]。
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceQueueSchedule.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({ apiFetch: vi.fn() }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({
      params: { tenant: 't1' },
      query: { workspace_id: 'ws9' },
    }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  const { default: WorkspaceQueueSchedule } = await import('./WorkspaceQueueSchedule.vue')

  const SNAP = {
    workspace_id: 'ws9',
    tenant_id: 't1',
    schedule_rhythm: {
      enabled: true,
      timezone: 'Asia/Shanghai',
      windows: [{ id: 'w0', daily_start: '09:00', daily_end: '11:00', max_queued_machines: 2, auto_close: false }],
      max_queued_machines: 2,
      auto_close: false,
      in_window: true,
    },
    in_window: true,
    window_message: '当前处于允许运行时段',
    queued_slots_used: 1,
    max_queued_machines: 2,
    members: [
      { task_id: 'ta', title: '顶层任务', depth: 0, status: 'queued', enqueued_at: '2026-08-24T00:00:00Z' },
      { task_id: 'tb', title: '子任务', depth: 1, status: 'deferred', enqueued_at: '2026-08-24T00:01:00Z' },
    ],
    recent_history: [
      {
        id: 'qsh_1',
        event_type: 'member_enqueued',
        message: '加入自动调度队列：顶层任务',
        created_at: '2026-08-28T00:00:00Z',
        task_id: 'ta',
        task_title: '顶层任务',
      },
    ],
  }

  function jsonResponse(body, { ok = true, status = 200 } = {}) {
    return {
      ok,
      status,
      headers: { get: () => null },
      json: async () => body,
    }
  }

  const traceIdOf = (wrapper, testid) => {
    const el = wrapper.get(`[data-testid="${testid}"]`).element
    for (const attr of el.attributes) {
      if (attr.name.toLowerCase() === 'data-traceid') return attr.value
    }
    return undefined
  }

  describe('WorkspaceQueueSchedule 页面', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      window.apiFetch = hoisted.apiFetch
    })
    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('URL workspace_id 优先解析 + 渲染状态条/节奏表单/队列成员', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()

      // /me/ 与 queue-schedule GET 各一次
      expect(hoisted.apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/accounts/users/me/'),
        expect.anything(),
      )
      expect(hoisted.apiFetch).toHaveBeenCalledWith(
        '/api/tenant/t1/workspace/ws9/queue-schedule/',
        expect.anything(),
      )

      expect(wrapper.get('[data-testid="schedule-status-bar"]').text()).toContain('允许运行时段内')
      expect(wrapper.get('[data-testid="schedule-status-bar"]').text()).toContain('1/2')
      expect(wrapper.get('[data-testid="schedule-history-card"]').text()).toContain('加入自动调度队列：顶层任务')
      const histLink = wrapper.get('[data-testid="schedule-history-task-ta"]')
      expect(histLink.element.tagName).toBe('A')
      expect(histLink.attributes('href')).toBe('/tenant/t1/workspace/ws9/task-detail/ta/')
      const enabled = wrapper.get('[data-testid="schedule-enabled"]').element
      expect(enabled.checked).toBe(true)
      expect(wrapper.get('[data-testid="queue-members-card"]').text()).toContain('顶层任务')
      expect(wrapper.get('[data-testid="member-status-ta"]').text()).toBe('排队中')
      expect(wrapper.get('[data-testid="member-status-tb"]').text()).toBe('等待时段')
      expect(wrapper.get('[data-testid="queue-join-toggle"]').text()).toContain('加入队列')

      const titleLink = wrapper.get('[data-testid="queue-member-title-ta"]')
      expect(titleLink.element.tagName).toBe('A')
      expect(titleLink.attributes('href')).toBe('/tenant/t1/workspace/ws9/task-detail/ta/')
      expect(titleLink.text()).toBe('顶层任务')
    })

    it('空标题仍渲染任务详情 a[href]，文本为（无标题）', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse({
        ...SNAP,
        members: [{ task_id: 'tz', title: '', depth: 0, status: 'queued', enqueued_at: '2026-08-24T00:00:00Z' }],
      }))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()
      const titleLink = wrapper.get('[data-testid="queue-member-title-tz"]')
      expect(titleLink.element.tagName).toBe('A')
      expect(titleLink.attributes('href')).toBe('/tenant/t1/workspace/ws9/task-detail/tz/')
      expect(titleLink.text()).toBe('（无标题）')
    })

    it('缺少 task_id 时标题为 span，不渲染 href="#"', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse({
        ...SNAP,
        members: [{ task_id: '', title: '无 ID 任务', depth: 0, status: 'queued', enqueued_at: '2026-08-24T00:00:00Z' }],
      }))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()
      const titleEl = wrapper.get('[data-testid="queue-member-title-unknown"]')
      expect(titleEl.element.tagName).toBe('SPAN')
      expect(titleEl.attributes('href')).toBeUndefined()
      expect(titleEl.text()).toBe('无 ID 任务')
      expect(wrapper.find('a[href="#"]').exists()).toBe(false)
    })

    it('无 recent_history 时展示调度历史空态', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse({ ...SNAP, recent_history: [] }))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()
      expect(wrapper.get('[data-testid="schedule-status-bar"]').exists()).toBe(true)
      expect(wrapper.get('[data-testid="schedule-history-empty"]').text()).toContain('尚无调度记录')
    })

    it('点击刷新会再发 GET queue-schedule', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()
      hoisted.apiFetch.mockClear()
      hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
      await wrapper.get('[data-testid="refresh-members"]').trigger('click')
      await flushPromises()
      expect(hoisted.apiFetch).toHaveBeenCalledWith(
        '/api/tenant/t1/workspace/ws9/queue-schedule/',
        expect.objectContaining({ method: 'GET' }),
      )
    })

    it('保存：PUT 后显示保存成功提示', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()
      hoisted.apiFetch.mockClear()

      hoisted.apiFetch.mockResolvedValue(jsonResponse(SNAP))
      await wrapper.get('[data-testid="save-schedule"]').trigger('click')
      await flushPromises()

      const [url, opts] = hoisted.apiFetch.mock.calls[0]
      expect(url).toBe('/api/tenant/t1/workspace/ws9/queue-schedule/')
      expect(opts.method).toBe('PUT')
      const body = JSON.parse(opts.body)
      expect(body.enabled).toBe(true)
      expect(body.windows[0].daily_start).toBe('09:00')
      expect(wrapper.get('[data-testid="schedule-saved-tip"]').text()).toContain('已保存')
    })

    it('加载失败 → 错误块挂 data-traceId', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse(
        { status: 'error', error: '加载工作空间排队调度失败', message: 'boom', trace_id: 'tr-xyz' },
        { ok: false, status: 500 },
      ))
      const wrapper = mount(WorkspaceQueueSchedule)
      await flushPromises()
      expect(traceIdOf(wrapper, 'schedule-error')).toBe('tr-xyz')
    })
  })
}
