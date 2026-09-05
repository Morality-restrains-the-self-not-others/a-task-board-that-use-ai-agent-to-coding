// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] taskPanelTodoApi.createComment.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('./apiUtils.js', () => ({
    apiFetch,
  }))

  const { createTaskComment, updateTaskComment } = await import('./taskPanelTodoApi.js')

  describe('createTaskComment', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue({ ok: true })
    })

    it('兼容纯文本 content', async () => {
      await createTaskComment('t1', 'task-1', 'hello')
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/tasks/task-1/comments/'),
        expect.objectContaining({
          body: JSON.stringify({ task: 'task-1', content: 'hello' }),
        }),
      )
    })

    it('支持 mentions payload（与任务详情提交一致）', async () => {
      await createTaskComment('t1', 'task-1', {
        content: 'run @node',
        mentions: [{ type: 'installed_image', id: 'img1', name: 'node' }],
      })
      expect(apiFetch).toHaveBeenCalledWith(
        expect.stringContaining('/tasks/task-1/comments/'),
        expect.objectContaining({
          body: JSON.stringify({
            task: 'task-1',
            content: 'run @node',
            mentions: [{ type: 'installed_image', id: 'img1', name: 'node' }],
          }),
        }),
      )
    })

    it('拒绝内容仅为任务编号的评论', async () => {
      await expect(createTaskComment('t1', 'task_abc', 'task_abc')).rejects.toThrow(
        '评论内容不能仅为任务编号',
      )
      expect(apiFetch).not.toHaveBeenCalled()
    })

    it('updateTaskComment: 评论 ID 为独立路径段（e105bad 迁移回归）', async () => {
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
      await updateTaskComment('t1', 'task-1', 'c1', '新内容')
      expect(apiFetch).toHaveBeenCalledWith(
        '/api/tasks/task-1/comments/c1/tenant_id/t1/',
        expect.objectContaining({ method: 'PUT' }),
      )
    })
  })
}
