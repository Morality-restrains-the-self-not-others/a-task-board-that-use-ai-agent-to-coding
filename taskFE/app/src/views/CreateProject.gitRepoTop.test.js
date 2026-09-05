// @vitest-environment jsdom
/**
 * CreateProject Git 仓库输入置顶：#gitRepo0 须在 #projectName 之前（便于仓库授权）。
 */
if (!process.env.VITEST) {
  console.log('[skip] CreateProject.gitRepoTop.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1' } }),
    useRouter: () => ({ push: vi.fn() }),
  }))

  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const CreateProject = (await import('./CreateProject.vue')).default

  function jsonResponse(body) {
    return { ok: true, status: 200, json: async () => body }
  }

  describe('CreateProject Git 仓库置顶', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/installed-images/')) return jsonResponse([])
        if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
        return jsonResponse({})
      })
    })

    it('#gitRepo0 在 DOM 中位于 #projectName 之前', async () => {
      const wrapper = mount(CreateProject, {
        global: {
          stubs: {
            ProjectTagsInput: { template: '<div class="stub-tags" />' },
            ProjectRunTemplatePanel: { template: '<div class="stub-run-template" />' },
            'router-link': { template: '<a><slot /></a>' },
          },
        },
      })
      await flushPromises()

      const git = wrapper.find('#gitRepo0').element
      const name = wrapper.find('#projectName').element
      expect(git).toBeTruthy()
      expect(name).toBeTruthy()
      // DOCUMENT_POSITION_FOLLOWING = 4
      expect(git.compareDocumentPosition(name) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    })
  })
}
