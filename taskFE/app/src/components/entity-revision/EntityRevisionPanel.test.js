// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] EntityRevisionPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: hoisted.apiFetchMock,
  }))

  const { default: EntityRevisionPanel } = await import('./EntityRevisionPanel.vue')

  function jsonResponse(body, { ok = true, status = 200, traceId = '' } = {}) {
    return {
      ok,
      status,
      headers: {
        get: (name) => {
          if (String(name).toLowerCase() === 'x-trace-id') return traceId
          return null
        },
      },
      json: async () => body,
    }
  }

  describe('EntityRevisionPanel', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
    })

    it('F1 渲染历史版本入口且 F7 未打开不请求', () => {
      const wrapper = mount(EntityRevisionPanel, {
        props: { listUrl: '/api/tasks/todos/tenant_id/t1/workspace_id/ws1/task1/revisions/' },
      })
      expect(wrapper.find('[data-testid=entity-revision-open]').exists()).toBe(true)
      expect(hoisted.apiFetchMock).not.toHaveBeenCalled()
    })

    it('F2 打开后 GET 列表并渲染行', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonResponse({
        results: [{ id: 'r1', version_num: 1, title: 'Hello', description: 'body', created_at: '2026-08-30T00:00:00Z', changed_fields: 'title,description' }],
        total: 1,
      }))
      const wrapper = mount(EntityRevisionPanel, {
        props: { listUrl: '/api/x/revisions/' },
      })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      expect(hoisted.apiFetchMock).toHaveBeenCalledTimes(1)
      expect(hoisted.apiFetchMock.mock.calls[0][0]).toContain('/revisions/')
      expect(wrapper.findAll('[data-testid=entity-revision-row]').length).toBe(1)
    })

    it('F3 点选版本 GET 详情并展示完整标题与正文', async () => {
      hoisted.apiFetchMock.mockImplementation(async (url) => {
        const path = String(url)
        if (path.endsWith('/r1/') || path.includes('/r1/')) {
          return jsonResponse({
            id: 'r1', version_num: 2, title: 'World', description: 'full text beyond list truncation', created_at: '2026-08-30T00:00:00Z',
          })
        }
        return jsonResponse({
          results: [{ id: 'r1', version_num: 2, title: 'World', description: 'truncated…', created_at: '2026-08-30T00:00:00Z' }],
          total: 1,
        })
      })
      const wrapper = mount(EntityRevisionPanel, { props: { listUrl: '/api/x/revisions/' } })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      await wrapper.find('[data-testid=entity-revision-row] button').trigger('click')
      await flushPromises()
      expect(hoisted.apiFetchMock.mock.calls.some((call) => String(call[0]).includes('/revisions/r1/'))).toBe(true)
      expect(wrapper.find('[data-testid=entity-revision-detail]').text()).toContain('World')
      expect(wrapper.find('[data-testid=entity-revision-detail]').text()).toContain('full text beyond list truncation')
    })

    it('F4 空列表文案', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonResponse({ results: [], total: 0 }))
      const wrapper = mount(EntityRevisionPanel, { props: { listUrl: '/api/x/revisions/' } })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-empty]').text()).toContain('尚无历史版本')
    })

    it('F5 失败节点带 data-traceId', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonResponse(
        { error: 'forbidden', trace_id: 'trace-abc' },
        { ok: false, status: 403, traceId: 'trace-abc' },
      ))
      const wrapper = mount(EntityRevisionPanel, { props: { listUrl: '/api/x/revisions/' } })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      const err = wrapper.find('[data-testid=entity-revision-error]')
      expect(err.exists()).toBe(true)
      expect(err.attributes('data-traceid') || err.attributes('data-traceId')).toBe('trace-abc')
    })

    it('F6 项目 name 字段展示', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonResponse({
        results: [{ id: 'r1', version_num: 1, name: 'Alpha', description: 'd', created_at: '2026-08-30T00:00:00Z' }],
        total: 1,
      }))
      const wrapper = mount(EntityRevisionPanel, {
        props: { listUrl: '/api/projects/tenant_id/t1/p1/revisions/', titleField: 'name' },
      })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-row]').text()).toContain('Alpha')
    })

    it('F8 列表仍在加载时也可点收起关闭面板，入口不得保持 disabled', async () => {
      let resolveFetch
      hoisted.apiFetchMock.mockImplementation(
        () => new Promise((resolve) => {
          resolveFetch = resolve
        }),
      )
      const wrapper = mount(EntityRevisionPanel, { props: { listUrl: '/api/x/revisions/' } })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(true)
      const openBtn = wrapper.find('[data-testid=entity-revision-open]')
      expect(openBtn.attributes('disabled')).toBeUndefined()
      expect(wrapper.find('[data-testid=entity-revision-close]').exists()).toBe(true)
      await wrapper.find('[data-testid=entity-revision-close]').trigger('click')
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(false)
      resolveFetch(jsonResponse({ results: [], total: 0 }))
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(false)
    })

    it('F9 单击展开、双击收起', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonResponse({ results: [], total: 0 }))
      const wrapper = mount(EntityRevisionPanel, { props: { listUrl: '/api/x/revisions/' } })
      const openBtn = wrapper.find('[data-testid=entity-revision-open]')
      await openBtn.trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(true)
      await openBtn.trigger('dblclick')
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(false)
    })

    it('F10 点击面板外收起', async () => {
      hoisted.apiFetchMock.mockResolvedValue(jsonResponse({ results: [], total: 0 }))
      const wrapper = mount(EntityRevisionPanel, {
        props: { listUrl: '/api/x/revisions/' },
        attachTo: document.body,
      })
      await wrapper.find('[data-testid=entity-revision-open]').trigger('click')
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(true)
      document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
      await flushPromises()
      expect(wrapper.find('[data-testid=entity-revision-panel]').exists()).toBe(false)
      wrapper.unmount()
    })
  })
}
