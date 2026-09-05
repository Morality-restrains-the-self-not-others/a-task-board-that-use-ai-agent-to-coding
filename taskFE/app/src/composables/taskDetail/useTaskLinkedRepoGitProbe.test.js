// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] useTaskLinkedRepoGitProbe.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { ref, nextTick } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  vi.mock('../../utils/repoOAuthAuthorizeUtils.js', () => ({
    resolveRepoOAuthProviderInfo: (url) => (String(url || '').includes('github.com') ? { provider: 'github' } : null),
  }))

  const { useTaskLinkedRepoGitProbe } = await import('./useTaskLinkedRepoGitProbe.js')

  describe('useTaskLinkedRepoGitProbe', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
    })

    it('POSTs validate-git-repos without throwing when tenant and repo URLs are set', async () => {
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [
            { url: 'https://github.com/demo/repo-a.git', token_status: 'token_error' },
          ],
        }),
      })
      const tenantId = ref('t1')
      const repoUrls = ref(['https://github.com/demo/repo-a.git'])
      const { gitRepoTokenStatus, gitRepoStatusLoading } = useTaskLinkedRepoGitProbe({
        tenantId,
        repoUrls,
      })
      await vi.waitFor(() => {
        expect(hoisted.apiFetchMock).toHaveBeenCalled()
      })
      const [url, opts] = hoisted.apiFetchMock.mock.calls[0]
      expect(url).toContain('/projects/validate-git-repos/')
      expect(opts.method).toBe('POST')
      const body = JSON.parse(opts.body)
      expect(body.probe_access).toBe(true)
      expect(body.urls).toEqual(['https://github.com/demo/repo-a.git'])
      await nextTick()
      await vi.waitFor(() => {
        expect(gitRepoTokenStatus.value['https://github.com/demo/repo-a.git']).toBe('token_error')
      })
      expect(gitRepoStatusLoading.value['https://github.com/demo/repo-a.git']).toBeFalsy()
    })

    it('does not POST when there is no tenant', async () => {
      const { fetchLinkedRepoGitProbe } = useTaskLinkedRepoGitProbe({
        tenantId: ref(''),
        repoUrls: ref(['https://github.com/demo/repo-a.git']),
      })
      await fetchLinkedRepoGitProbe()
      expect(hoisted.apiFetchMock).not.toHaveBeenCalled()
    })

    it('records traceId on probe failure and keeps going', async () => {
      const err = new Error('network')
      err.name = 'TimeoutError'
      err.traceId = 'trace-probe-1'
      hoisted.apiFetchMock.mockRejectedValue(err)
      const { gitRepoProbeErrorByUrl, gitRepoProbeTraceIdByUrl, fetchLinkedRepoGitProbe } = useTaskLinkedRepoGitProbe({
        tenantId: ref('t1'),
        repoUrls: ref(['https://github.com/demo/repo-a.git']),
      })
      await fetchLinkedRepoGitProbe()
      expect(gitRepoProbeErrorByUrl.value['https://github.com/demo/repo-a.git']).toBeTruthy()
      expect(gitRepoProbeTraceIdByUrl.value['https://github.com/demo/repo-a.git']).toBe('trace-probe-1')
    })
  })
}
