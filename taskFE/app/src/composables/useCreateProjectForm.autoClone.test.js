// @vitest-environment jsdom
/**
 * useCreateProjectForm：auto_clone_nested_repos 默认 true，提交透传。
 */
if (!process.env.VITEST) {
  console.log('[skip] useCreateProjectForm.autoClone.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  const hoisted = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    push: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => hoisted.apiFetch(...args),
  }))

  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: vi.fn(),
  }))

  const { useCreateProjectForm } = await import('./useCreateProjectForm.js')

  function jsonResponse(body, { ok = true, status = 201 } = {}) {
    return { ok, status, json: async () => body }
  }

  describe('useCreateProjectForm auto_clone_nested_repos', () => {
    beforeEach(() => {
      hoisted.apiFetch.mockReset()
      hoisted.push.mockReset()
    })

    it('defaults auto_clone_nested_repos to true and posts false when unchecked', async () => {
      let posted
      hoisted.apiFetch.mockImplementation(async (url, opts = {}) => {
        const u = String(url)
        if (u.includes('/installed-images/') || u.includes('/workspaces/')) {
          return jsonResponse([])
        }
        if (opts.method === 'POST') {
          posted = JSON.parse(opts.body)
          return jsonResponse({ id: 'proj_1' })
        }
        return jsonResponse({})
      })

      const gitRepoRows = ref([{ id: 1, url: 'https://github.com/o/r.git', cloneAlias: '' }])
      const form = useCreateProjectForm({
        route: { params: { tenant: 't1' } },
        router: { push: hoisted.push },
        gitRepoRows,
        trimmedGitRepoPayload: () => [{ url: 'https://github.com/o/r.git', clone_alias: '' }],
        duplicateCloneAliasError: () => '',
        gitReposHaveValidUrls: () => true,
        gitReposPendingValidation: ref(false),
        appendGitRepoFormErrors: () => {},
        runTemplatePanelRef: ref({ buildPayload: () => ({}) }),
      })

      expect(form.formData.value.auto_clone_nested_repos).toBe(true)
      expect(form.hasAnyGitRepoUrl.value).toBe(true)

      form.formData.value.name = 'n'
      form.formData.value.description = 'd'
      form.formData.value.workspace = 'ws1'
      form.formData.value.auto_clone_nested_repos = false

      await form.submitForm()

      expect(posted?.auto_clone_nested_repos).toBe(false)
      expect(hoisted.push).toHaveBeenCalled()
    })
  })
}
