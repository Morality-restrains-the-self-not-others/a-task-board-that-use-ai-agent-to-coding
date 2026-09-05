// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkPanelTaskSearch.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  const { default: WorkPanelTaskSearch } = await import('./WorkPanelTaskSearch.vue')
  const apiUtils = await import('../utils/apiUtils.js')

  describe('WorkPanelTaskSearch', () => {
    beforeEach(() => {
      vi.spyOn(apiUtils, 'apiFetch').mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('company_members')) {
          return {
            ok: true,
            json: async () => ({
              members: [{ id: 'm-bob', member_name: 'Bob' }],
            }),
          }
        }
        if (u.includes('tasks/search')) {
          return {
            ok: true,
            json: async () => ({
              results: [
                {
                  id: 'task-1',
                  title: 'Alpha',
                  workspace_id: 'ws1',
                  owner: 'm-bob',
                  assignees: [],
                },
              ],
            }),
          }
        }
        return { ok: false, status: 404, json: async () => ({}) }
      })
    })

    afterEach(() => {
      vi.restoreAllMocks()
      vi.useRealTimers()
    })

    it('可见时渲染输入框，搜索后可跳转工作面板与任务', async () => {
      vi.useFakeTimers()
      const wrapper = mount(WorkPanelTaskSearch, {
        props: { visible: true, tenantId: '850', accessCode: '' },
      })
      expect(wrapper.find('[data-testid="work-panel-task-search-input"]').exists()).toBe(true)

      const input = wrapper.get('[data-testid="work-panel-task-search-input"]')
      await input.setValue('Alpha')
      await vi.advanceTimersByTimeAsync(300)
      await flushPromises()

      expect(wrapper.findAll('[data-testid="work-panel-task-search-hit"]').length).toBe(1)

      const openTask = wrapper.get('[data-testid="work-panel-task-search-open-task"]')
      expect(openTask.element.tagName).toBe('A')
      expect(openTask.attributes('href')).toBe('/tenant/850/workspace/ws1/task-detail/task-1/')

      const openPanel = wrapper.get('[data-testid="work-panel-task-search-open-panel"]')
      expect(openPanel.element.tagName).toBe('A')
      expect(openPanel.attributes('href')).toBe('/tenant/850/work-panel/?workspace_id=ws1')

      await openTask.trigger('click')
      expect(wrapper.find('[data-testid="work-panel-task-search-dropdown"]').exists()).toBe(true)
    })

    it('粘贴评论容器名时请求 q 为规范任务 id', async () => {
      vi.useFakeTimers()
      const wrapper = mount(WorkPanelTaskSearch, {
        props: { visible: true, tenantId: '850', accessCode: '' },
      })
      const input = wrapper.get('[data-testid="work-panel-task-search-input"]')
      expect(input.attributes('placeholder')).toContain('评论容器')
      await input.setValue('task_877071722828820480_cmt_877071748669927424')
      await vi.advanceTimersByTimeAsync(300)
      await flushPromises()

      const searchCalls = apiUtils.apiFetch.mock.calls
        .map((args) => String(args[0]))
        .filter((u) => u.includes('tasks/search'))
      expect(searchCalls.length).toBeGreaterThan(0)
      const qs = new URL(searchCalls[searchCalls.length - 1], 'http://local.test').searchParams
      expect(qs.get('q')).toBe('task_877071722828820480')
      expect(qs.get('q')).not.toContain('_cmt_')
    })
  })
}
