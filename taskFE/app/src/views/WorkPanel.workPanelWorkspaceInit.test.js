// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] workPanelWorkspaceInit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { resolveWorkPanelInitialWorkspace } = await import('../utils/workPanelWorkspaceInit.js')

  describe('resolveWorkPanelInitialWorkspace', () => {
    it('URL workspace_id 与 /me/ 不一致时以 URL 为准（不回退用户默认工作空间）', () => {
      const ws = resolveWorkPanelInitialWorkspace({
        urlWorkspaceId: 'ws_B',
        userWorkspace: { id: 'ws_A', name: '用户默认工作空间' },
      })
      expect(ws.id).toBe('ws_B')
      expect(ws.name).toBe('工作空间')
    })

    it('URL workspace_id 与 /me/ 一致时复用完整用户工作空间对象（含展示名）', () => {
      const ws = resolveWorkPanelInitialWorkspace({
        urlWorkspaceId: 'ws_A',
        userWorkspace: { id: 'ws_A', name: '用户默认工作空间' },
      })
      expect(ws.id).toBe('ws_A')
      expect(ws.name).toBe('用户默认工作空间')
    })

    it('URL 未指定时回退 /me/ current_workspace', () => {
      const ws = resolveWorkPanelInitialWorkspace({
        urlWorkspaceId: null,
        userWorkspace: { id: 'ws_A', name: '用户默认工作空间' },
      })
      expect(ws.id).toBe('ws_A')
      expect(ws.name).toBe('用户默认工作空间')
    })

    it('URL 与 /me/ 都无可用工作空间时返回默认占位', () => {
      const ws = resolveWorkPanelInitialWorkspace({ urlWorkspaceId: null, userWorkspace: null })
      expect(ws.id).toBeNull()
      expect(ws.name).toBe('默认工作空间')
    })

    it('数字 id 与字符串 id 按值比较（URL 查询参数恒为字符串；真实 id 超 2^53 故后端以字符串下发）', () => {
      const ws = resolveWorkPanelInitialWorkspace({
        urlWorkspaceId: '8755882835',
        userWorkspace: { id: 8755882835, name: '团队工作空间' },
      })
      expect(ws.id).toBe(8755882835)
      expect(ws.name).toBe('团队工作空间')
    })
  })
}
