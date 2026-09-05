// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchLayerFileContent.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoisted = vi.hoisted(() => ({ apiFetch: vi.fn() }))

  vi.mock('./apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  const { fetchLayerFileContentWithPrefixFallback, fetchLayerRepoPathPrefixes } = await import(
    './taskDetailFetchLayerFileContent.js'
  )

  function jsonResponse(body, { ok = true, status = 200 } = {}) {
    return { ok, status, json: async () => body, headers: { get: () => null } }
  }

  describe('fetchLayerFileContent comment_id', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
    })

    it('appends comment_id on file content GET', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse({ content: 'ok' }))
      const result = await fetchLayerFileContentWithPrefixFallback(
        {
          tenantId: 't1',
          workspaceId: 'ws1',
          taskId: 'task1',
          layerId: 'L1',
          commentId: 'cmt_9',
        },
        'README.md',
        [],
      )
      expect(result.ok).toBe(true)
      const url = hoisted.apiFetch.mock.calls[0][0]
      expect(url).toContain('/comment_id/cmt_9/')
      expect(url).toContain('path=README.md')
      expect(url).not.toMatch(/[?&]comment_id=/)
    })

    it('appends comment_id on git-repo-identities GET', async () => {
      hoisted.apiFetch.mockResolvedValue(jsonResponse({ repos: [{ rel_prefix: 'ram-work' }] }))
      const prefixes = await fetchLayerRepoPathPrefixes({
        tenantId: 't1',
        workspaceId: 'ws1',
        taskId: 'task1',
        layerId: 'L1',
        commentId: 'cmt_9',
      })
      expect(prefixes).toEqual(['ram-work'])
      expect(hoisted.apiFetch.mock.calls[0][0]).toContain('/comment_id/cmt_9/')
    })
  })
}
