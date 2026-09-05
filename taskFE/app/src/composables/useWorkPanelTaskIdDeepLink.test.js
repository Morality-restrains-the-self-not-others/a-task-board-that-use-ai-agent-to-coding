// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] useWorkPanelTaskIdDeepLink.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { ref } = await import('vue')
  const { useWorkPanelTaskIdDeepLink } = await import('./useWorkPanelTaskIdDeepLink.js')

  describe('useWorkPanelTaskIdDeepLink', () => {
    it('todos 就绪后按 task_id 打开任务', async () => {
      const openTask = vi.fn()
      const todos = ref([])
      const route = { query: { task_id: 't1' } }
      const router = { replace: vi.fn(async () => {}) }

      useWorkPanelTaskIdDeepLink({ route, router, todos, openTask })
      expect(openTask).not.toHaveBeenCalled()

      todos.value = [{ id: 't1', title: 'X' }]
      // watch 同步触发（immediate + deep）
      await Promise.resolve()
      expect(openTask).toHaveBeenCalledWith(expect.objectContaining({ id: 't1' }))
    })
  })
}
