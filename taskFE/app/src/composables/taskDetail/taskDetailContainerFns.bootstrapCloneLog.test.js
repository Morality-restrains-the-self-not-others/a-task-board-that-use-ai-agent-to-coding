// @vitest-environment node
/**
 * 引导克隆日志：凭证不齐时须回调失败，避免任务关联区无限「等待可写层」。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailContainerFns.bootstrapCloneLog.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { fetchContainerBootstrapCloneLog } = await import('./taskDetailContainerFns.js')

  function makeDeps(overrides = {}) {
    return {
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      displayComments: ref([{ id: 'cmt-1', commentKind: 'user' }]),
      activeContainerAgentId: ref(''),
      containerEndpointRegistered: ref(true),
      onBootstrapCloneLogUpdate: vi.fn(),
      onBootstrapCloneLogFailure: vi.fn(),
      ...overrides,
    }
  }

  describe('fetchContainerBootstrapCloneLog credentials failure', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('calls failure callback when error_code is REPO_CLONE_CREDENTIALS_INCOMPLETE', async () => {
      const failText =
        'repo-clone-credentials 未返回完整 repo_clone_credentials；缺失仓库(1): https://github.com/ruandao/somanyad'
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          error_code: 'REPO_CLONE_CREDENTIALS_INCOMPLETE',
          text: failText,
        }),
      })
      const deps = makeDeps()
      await fetchContainerBootstrapCloneLog(deps)
      expect(deps.onBootstrapCloneLogUpdate).toHaveBeenCalled()
      expect(deps.onBootstrapCloneLogFailure).toHaveBeenCalled()
      const msg = String(deps.onBootstrapCloneLogFailure.mock.calls[0][0] || '')
      expect(msg).toContain('Git 授权未齐')
    })

    it('calls failure callback when payload has only error_code', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ error_code: 'REPO_CLONE_CREDENTIALS_INCOMPLETE' }),
      })
      const deps = makeDeps()
      await fetchContainerBootstrapCloneLog(deps)
      expect(deps.onBootstrapCloneLogUpdate).not.toHaveBeenCalled()
      expect(deps.onBootstrapCloneLogFailure).toHaveBeenCalled()
    })

    it('does not treat successful clone log as failure', async () => {
      apiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({
          text: '【项目克隆】克隆完成。',
        }),
      })
      const deps = makeDeps()
      await fetchContainerBootstrapCloneLog(deps)
      expect(deps.onBootstrapCloneLogUpdate).toHaveBeenCalled()
      expect(deps.onBootstrapCloneLogFailure).not.toHaveBeenCalled()
    })
  })
}
