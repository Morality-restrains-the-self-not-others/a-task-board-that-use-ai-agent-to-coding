// @vitest-environment jsdom
/**
 * CreateProject：仓库不可访问时须提示「是否现在去授权」并展示 OAuth 授权按钮。
 */
if (!process.env.VITEST) {
  console.log('[skip] CreateProject.oauth-prompt.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const { setGitOAuthProviderCatalogForTests } = await import('../utils/repoOAuthAuthorizeUtils.js')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))

  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const CreateProject = (await import('./CreateProject.vue')).default

  function jsonResponse(body, { ok = true, status = 200 } = {}) {
    return { ok, status, json: async () => body }
  }

  describe('CreateProject 仓库不可访问 OAuth 引导', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      setGitOAuthProviderCatalogForTests([
        {
          provider: 'github',
          service_provider: 'default',
          website: 'https://github.com',
        },
      ])
      vi.useFakeTimers()
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/installed-images/')) return jsonResponse([])
        if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
        if (u.includes('/git-oauth/providers/')) {
          return jsonResponse({
            providers: [
              { provider: 'github', service_provider: 'default', website: 'https://github.com' },
            ],
          })
        }
        if (u.includes('/validate-git-repos/')) {
          return jsonResponse({
            results: [
              {
                url: 'https://github.com/private-org/private-repo',
                repo_url: 'https://github.com/private-org/private-repo',
                is_accessible: false,
                token_status: 'not_bound',
                message: '',
              },
            ],
          })
        }
        return jsonResponse({})
      })
    })

    afterEach(() => {
      setGitOAuthProviderCatalogForTests(null)
      vi.useRealTimers()
    })

    it('不可访问时显示「是否现在去授权」与 OAuth 授权按钮', async () => {
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

      await wrapper.find('#gitRepo0').setValue('https://github.com/private-org/private-repo')
      await vi.advanceTimersByTimeAsync(600)
      await flushPromises()

      expect(wrapper.text()).toContain('该仓库无法访问，后续需要授权后才能访问')
      expect(wrapper.find('[data-testid="repo-oauth-authorize-prompt"]').text()).toContain(
        '是否现在去授权',
      )
      const oauthBtn = wrapper.find('[data-testid="repo-oauth-authorize-button"]')
      expect(oauthBtn.exists()).toBe(true)
      expect(oauthBtn.text()).toContain('OAuth 授权')
    })

    it('校验 HTTP 失败时错误节点带 data-traceId 且提供重试', async () => {
      hoisted.apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/installed-images/')) return jsonResponse([])
        if (u.includes('/workspaces/')) return jsonResponse([{ id: 'ws1', name: 'WS1' }])
        if (u.includes('/git-oauth/providers/')) {
          return jsonResponse({
            providers: [
              { provider: 'github', service_provider: 'default', website: 'https://github.com' },
            ],
          })
        }
        if (u.includes('/validate-git-repos/')) {
          return {
            ok: false,
            status: 502,
            traceId: 'trace-create-validate-1',
            json: async () => ({
              error: '服务暂时不可用，请稍后重试',
              message: '服务暂时不可用，请稍后重试',
            }),
          }
        }
        return jsonResponse({})
      })
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
      await wrapper.find('#gitRepo0').setValue('https://github.com/org/repo.git')
      await vi.advanceTimersByTimeAsync(600)
      await flushPromises()

      const err = wrapper.find('[data-testid="git-repo-validate-error"]')
      expect(err.exists()).toBe(true)
      expect(err.text()).toContain('服务暂时不可用，请稍后重试')
      expect(err.attributes('data-traceid')).toBe('trace-create-validate-1')
      expect(wrapper.find('[data-testid="git-repo-validate-retry"]').exists()).toBe(true)
    })
  })
}
