// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] serverStartupStatusPoll.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  buildServerStartupStatusPollUrl,
  shouldApplyPolledStartupStatus,
  shouldPollServerStartupStatus,
} = await import('./serverStartupStatusPoll.js')

describe('serverStartupStatusPoll', () => {
  describe('shouldPollServerStartupStatus', () => {
    it('polls when isServerStarting is true regardless of SSE state', () => {
      expect(shouldPollServerStartupStatus({ isServerStarting: true, sseLive: false })).toBe(true)
      // SSE 连接时也轮询（慢速兜底），防止 success 事件丢失导致永久卡「启动中」
      expect(shouldPollServerStartupStatus({ isServerStarting: true, sseLive: true })).toBe(true)
      expect(shouldPollServerStartupStatus({ isServerStarting: false, sseLive: false })).toBe(false)
      expect(shouldPollServerStartupStatus({ isServerStarting: false, sseLive: true })).toBe(false)
    })
  })

  describe('shouldApplyPolledStartupStatus', () => {
    it('applies when polled progress is higher', () => {
      expect(shouldApplyPolledStartupStatus(5, 60)).toBe(true)
      expect(shouldApplyPolledStartupStatus(80, 60)).toBe(false)
    })
  })

  describe('buildServerStartupStatusPollUrl', () => {
    it('includes task_id and optional event_id', () => {
      const url = buildServerStartupStatusPollUrl({
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task1',
        eventId: 'ev1',
      })
      expect(url).toContain('/api/cloud/compute/server-startup-status/tenant_id/t1/workspace_id/w1')
      expect(url).toContain('task_id=task1')
      expect(url).toContain('event_id=ev1')
    })

    it('OPT-20260810-036: 评论级启动时携带 comment_id，供后端 loadLatestStartEvent 过滤', () => {
      const url = buildServerStartupStatusPollUrl({
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task1',
        eventId: 'ev1',
        commentId: 'c1',
      })
      expect(url).toContain('/comment_id/c1/')
      expect(url).not.toMatch(/[?&]comment_id=/)
    })

    it('comment_id 为空时不附加参数', () => {
      const url = buildServerStartupStatusPollUrl({
        tenantId: 't1',
        workspaceId: 'w1',
        taskId: 'task1',
        commentId: '  ',
      })
      expect(url).not.toContain('comment_id')
    })
  })
})
}
