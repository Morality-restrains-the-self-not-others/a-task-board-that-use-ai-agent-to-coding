// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] useCreateTaskGitIdentities.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { defineComponent, reactive } = await import('vue')
  const { afterEach, beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  vi.mock('../utils/cookieUtils.js', () => ({
    getCookie: () => 'user-1',
  }))

  const { useCreateTaskGitIdentities } = await import('./useCreateTaskGitIdentities.js')

  const Harness = defineComponent({
    name: 'UseCreateTaskGitIdentitiesHarness',
    props: {
      draft: { type: Object, required: true },
      tenantId: { type: String, default: 't1' },
      enabled: { type: Boolean, default: true },
      repoUrls: { type: Array, default: () => [] },
    },
    setup(props) {
      return useCreateTaskGitIdentities({
        editingTask: () => props.draft,
        projects: () => [],
        tenantId: () => props.tenantId,
        enabled: () => props.enabled,
        repoUrls: () => props.repoUrls,
      })
    },
    template: '<div />',
  })

  describe('useCreateTaskGitIdentities fork repoUrls', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      localStorage.setItem('currentUserId', 'user-1')
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => null },
        json: async () => ({
          identities: [{
            id: 'gid-current',
            git_user_name: 'Ann',
            git_user_email: 'ann@example.com',
            is_default: true,
          }],
        }),
      })
    })

    afterEach(() => {
      localStorage.clear()
    })

    it('applies current-user default identity to fork repo urls', async () => {
      const draft = reactive({ repo_identities: [] })
      const wrapper = mount(Harness, {
        props: {
          draft,
          enabled: true,
          repoUrls: ['https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad'],
        },
      })
      await flushPromises()
      await flushPromises()
      expect(draft.repo_identities).toEqual([{
        repo_url: 'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
        git_identity_id: 'gid-current',
      }])
      expect(wrapper.vm.gitIdentityIdForUrl(
        'https://gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad',
      )).toBe('gid-current')
      wrapper.unmount()
    })

    it('does not copy identities when disabled (non auto-run fork path)', async () => {
      const draft = reactive({ repo_identities: [] })
      const wrapper = mount(Harness, {
        props: {
          draft,
          enabled: false,
          repoUrls: ['https://gitlab.example/a.git'],
        },
      })
      await flushPromises()
      expect(draft.repo_identities).toEqual([])
      wrapper.unmount()
    })
  })
}
