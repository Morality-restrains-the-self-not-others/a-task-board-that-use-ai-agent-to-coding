// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useCreateTaskRepoOAuth.test.js requires vitest runtime')
} else {
  const { flushPromises, mount } = await import('@vue/test-utils')
  const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')
  const { defineComponent, h, ref } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    userId: 'user-1',
    sessionGrant: false,
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => hoisted.userId,
  }))

  vi.mock('../utils/grantTicketSession.js', () => ({
    hasSessionGrantForRepo: () => hoisted.sessionGrant,
    rememberGrantTicketFromSearch: () => '',
  }))

  const { useCreateTaskRepoOAuth } = await import('./useCreateTaskRepoOAuth.js')

  function mountOAuth(opts) {
    let api
    const Host = defineComponent({
      setup() {
        api = useCreateTaskRepoOAuth(opts)
        return api
      },
      render() {
        return h('div')
      },
    })
    const wrapper = mount(Host)
    return { wrapper, api }
  }

  describe('useCreateTaskRepoOAuth', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      hoisted.userId = 'user-1'
      hoisted.sessionGrant = false
    })

    afterEach(() => {
      vi.clearAllMocks()
    })

    it('marks repo bound when validate-git-repos returns token_available', async () => {
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [{ url: 'https://github.com/a/b.git', token_status: 'token_available' }],
        }),
      })
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => 't1',
        editingTask: () => ({ projectSelections: [{ projectId: 'p1' }], auto_run: true }),
        projects: () => [{ id: 'p1', git_repos: ['https://github.com/a/b.git'] }],
      })
      await flushPromises()
      expect(api.boundByUrl.value['https://github.com/a/b.git']).toBe(true)
      expect(api.blockedReason.value).toBe('')
      const body = JSON.parse(hoisted.apiFetchMock.mock.calls[0][1].body)
      expect(body.probe_access).toBe(false)
      expect(body.project_id).toBe('p1')
      wrapper.unmount()
    })

    it('keeps L1-only not_bound unbound', async () => {
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [{ url: 'https://github.com/a/b.git', token_status: 'not_bound' }],
        }),
      })
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => 't1',
        editingTask: () => ({ projectSelections: [{ projectId: 'p1' }], auto_run: true }),
        projects: () => [{ id: 'p1', git_repos: ['https://github.com/a/b.git'] }],
      })
      await flushPromises()
      expect(api.boundByUrl.value['https://github.com/a/b.git']).toBe(false)
      expect(api.blockedReason.value).toMatch(/项目详情/)
      wrapper.unmount()
    })

    it('accepts session grant_ticket without calling validate-git-repos', async () => {
      hoisted.sessionGrant = true
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => 't1',
        editingTask: () => ({ projectSelections: [{ projectId: 'p1' }], auto_run: true }),
        projects: () => [{ id: 'p1', git_repos: ['https://github.com/a/b.git'] }],
      })
      await flushPromises()
      expect(api.boundByUrl.value['https://github.com/a/b.git']).toBe(true)
      expect(hoisted.apiFetchMock).not.toHaveBeenCalled()
      wrapper.unmount()
    })

    it('treats project list git_repos_status token_available as bound without validate POST', async () => {
      const url = 'https://github.com/a/b.git'
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => 't1',
        editingTask: () => ({ projectSelections: [{ projectId: 'p1' }], auto_run: true }),
        projects: () => [{
          id: 'p1',
          git_repos: [url],
          git_repos_status: [{ repo_url: url, token_status: 'token_available' }],
        }],
      })
      await flushPromises()
      expect(api.boundByUrl.value[url]).toBe(true)
      expect(api.blockedReason.value).toBe('')
      expect(hoisted.apiFetchMock).not.toHaveBeenCalled()
      wrapper.unmount()
    })

    it('falls back to validate POST when list git_repos_status is not_applicable', async () => {
      const url = 'https://github.com/a/b.git'
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [{ url, token_status: 'token_available' }],
        }),
      })
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => 't1',
        editingTask: () => ({ projectSelections: [{ projectId: 'p1' }], auto_run: true }),
        projects: () => [{
          id: 'p1',
          git_repos: [url],
          git_repos_status: [{ repo_url: url, token_status: 'not_applicable' }],
        }],
      })
      await flushPromises()
      expect(api.boundByUrl.value[url]).toBe(true)
      expect(hoisted.apiFetchMock).toHaveBeenCalledTimes(1)
      wrapper.unmount()
    })

    it('marks object git_repos url bound when validate returns token_available', async () => {
      const url = 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git'
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({
          results: [{ url, token_status: 'token_available' }],
        }),
      })
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => 't1',
        editingTask: () => ({ projectSelections: [{ projectId: 'p1' }], auto_run: true }),
        projects: () => [{ id: 'p1', git_repos: [{ url, clone_alias: 'demo' }] }],
      })
      await flushPromises()
      expect(api.boundByUrl.value[url]).toBe(true)
      expect(api.blockedReason.value).toBe('')
      wrapper.unmount()
    })

    it('does not let a stale not_bound response overwrite token_available', async () => {
      const url = 'https://github.com/a/b.git'
      let resolveFirst
      const first = new Promise((resolve) => {
        resolveFirst = resolve
      })
      const tenantId = ref('t-old')
      const projectId = ref('p-old')
      hoisted.apiFetchMock.mockImplementation(async (_path, opts) => {
        const body = JSON.parse(opts.body)
        if (body.project_id === 'p-old') {
          await first
          return {
            ok: true,
            json: async () => ({ results: [{ url, token_status: 'not_bound' }] }),
          }
        }
        return {
          ok: true,
          json: async () => ({ results: [{ url, token_status: 'token_available' }] }),
        }
      })
      const { wrapper, api } = mountOAuth({
        enabled: () => true,
        tenantId: () => tenantId.value,
        editingTask: () => ({
          projectSelections: [{ projectId: projectId.value }],
          auto_run: true,
        }),
        projects: () => [
          { id: 'p-old', git_repos: [url] },
          { id: 'p-new', git_repos: [url] },
        ],
      })
      await flushPromises()
      tenantId.value = 't-new'
      projectId.value = 'p-new'
      await flushPromises()
      resolveFirst()
      await flushPromises()
      expect(api.boundByUrl.value[url]).toBe(true)
      wrapper.unmount()
    })
  })
}
