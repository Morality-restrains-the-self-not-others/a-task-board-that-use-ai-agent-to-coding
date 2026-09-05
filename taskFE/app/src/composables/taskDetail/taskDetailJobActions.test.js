// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailJobActions.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))
  vi.mock('../../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { showRequestError } = await import('../../utils/requestErrorDisplay.js')
  const { callLayerGraphJobAction, callLayerGraphLayerDelete } = await import('./taskDetailJobActions.js')

  function makeDeps(overrides = {}) {
    return {
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      effectiveTaskId: ref('task_1'),
      commentId: ref('cmt-exec'),
      layerGraphBusyActionKey: ref(''),
      markContainerTransportUnreachableIfForwardingFailed: vi.fn(),
      markContainerTransportOk: vi.fn(),
      refreshZTreeExecutionLog: vi.fn().mockResolvedValue(undefined),
      ...overrides,
    }
  }

  describe('callLayerGraphJobAction comment_id', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
      showRequestError.mockReset()
    })

    it('403 {message} 时经 showRequestError 展示业务文案并带 traceId', async () => {
      const headers = new Map([['X-Trace-Id', 'abc-trace-123']])
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        headers,
        _errorData: { message: '镜像要求与实例系统支持不匹配' },
      })
      await callLayerGraphJobAction('job-1', 'redo', '重做', makeDeps())
      expect(showRequestError).toHaveBeenCalledTimes(1)
      const [msg, source] = showRequestError.mock.calls[0]
      expect(msg).toContain('重做失败')
      expect(msg).toContain('镜像要求与实例系统支持不匹配')
      expect(source.headers).toBe(headers)
    })

    it('删除层级 403 时经 showRequestError 展示删除失败文案', async () => {
      apiFetch.mockResolvedValue({
        ok: false,
        status: 403,
        headers: new Map(),
        _errorData: { message: '该层级仍被引用' },
      })
      const ok = await callLayerGraphLayerDelete('layer-9', makeDeps())
      expect(ok).toBe(false)
      expect(showRequestError).toHaveBeenCalledTimes(1)
      expect(showRequestError.mock.calls[0][0]).toContain('删除失败：该层级仍被引用')
    })

    it('does not fetch without comment_id', async () => {
      const deps = makeDeps({ commentId: ref(''), displayComments: ref([]) })
      const ok = await callLayerGraphJobAction('job-1', 'redo', '重做', deps)
      expect(ok).toBe(false)
      expect(apiFetch).not.toHaveBeenCalled()
      expect(deps.layerGraphBusyActionKey.value).toBe('')
    })

    it('uses funcName-first URL with comment_id and path task_id', async () => {
      await callLayerGraphJobAction('job-1', 'redo', '重做', makeDeps())
      const url = String(apiFetch.mock.calls[0][0])
      expect(url).toContain('/api/cloud/compute/container-job-redo/tenant_id/t1/')
      expect(url).toContain('/task_id/task_1/')
      expect(url).toContain('/comment_id/cmt-exec/')
      expect(url).toContain('job_id=job-1')
      const init = apiFetch.mock.calls[0][1]
      expect(JSON.parse(init.body).comment_id).toBe('cmt-exec')
    })

    it('does not delete a layer without comment_id', async () => {
      const { callLayerGraphLayerDelete } = await import('./taskDetailJobActions.js')
      const deps = makeDeps({ commentId: ref(''), displayComments: ref([]) })
      const ok = await callLayerGraphLayerDelete('layer-1', deps)
      expect(ok).toBe(false)
      expect(apiFetch).not.toHaveBeenCalled()
    })
  })
}
