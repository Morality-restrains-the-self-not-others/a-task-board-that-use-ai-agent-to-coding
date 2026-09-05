// @vitest-environment node
/**
 * 两评论绑不同实例时，compute 转发 URL 与 POST body 必须带对应 comment_id。
 */
if (!process.env.VITEST) {
  console.log('[skip] containerComputeRequest.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const {
    containerComputeFuncFirstUrl,
    containerComputeKvLastUrl,
    postContainerCompute,
    getContainerCompute,
  } = await import('./containerComputeRequest.js')

  describe('containerComputeRequest comment_id', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockResolvedValue({ ok: true, json: async () => ({}) })
    })

    it('action 首尾斜杠被去掉且路径段仍正确', () => {
      const u = containerComputeKvLastUrl('t', 'w', 'k', '/container-job-redo/', 'cmt-a')
      expect(u).toContain('/container-job-redo/')
      expect(u).not.toContain('//container-job-redo')
    })

    it('两 comment_id 生成不同 path 段', () => {
      const a = containerComputeFuncFirstUrl('t', 'w', 'k', 'container-layer-graph', 'cmt-a')
      const b = containerComputeFuncFirstUrl('t', 'w', 'k', 'container-layer-graph', 'cmt-b')
      expect(a).toContain('/comment_id/cmt-a/')
      expect(b).toContain('/comment_id/cmt-b/')
      expect(a).not.toMatch(/[?&]comment_id=/)
      expect(a).not.toBe(b)
      const kvA = containerComputeKvLastUrl('t', 'w', 'k', 'container-job-redo', 'cmt-a', 'job_id=J1')
      expect(kvA).toContain('/comment_id/cmt-a/')
      expect(kvA).toContain('job_id=J1')
      expect(kvA).not.toMatch(/[?&]comment_id=/)
    })

    it('GET/POST 同时带 path 与 body comment_id', async () => {
      const deps = {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('ws1'),
        effectiveTaskId: ref('task1'),
        displayComments: ref([{ id: 'cmt-a', commentKind: 'ai' }]),
        activeContainerAgentId: ref(''),
      }
      await getContainerCompute(deps, 'container-layer-graph')
      await postContainerCompute(deps, 'container-layer-git-commit', { layer_id: 'L1' })
      expect(String(apiFetch.mock.calls[0][0])).toContain('/comment_id/cmt-a/')
      expect(String(apiFetch.mock.calls[1][0])).toContain('/comment_id/cmt-a/')
      expect(String(apiFetch.mock.calls[0][0])).not.toMatch(/[?&]comment_id=/)
      expect(JSON.parse(apiFetch.mock.calls[1][1].body)).toMatchObject({
        layer_id: 'L1',
        comment_id: 'cmt-a',
      })
    })

    it('切换评论后 POST 打到另一 comment_id', async () => {
      const deps = {
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('ws1'),
        effectiveTaskId: ref('task1'),
        displayComments: ref([{ id: 'cmt-b', commentKind: 'ai' }]),
        activeContainerAgentId: ref(''),
      }
      await postContainerCompute(deps, 'container-layer-git-push', { layer_id: 'L2' })
      expect(String(apiFetch.mock.calls[0][0])).toContain('/comment_id/cmt-b/')
      expect(JSON.parse(apiFetch.mock.calls[0][1].body).comment_id).toBe('cmt-b')
    })
  })
}
