// @vitest-environment jsdom
/**
 * $镜像 提交并运行：未绑定 Git OAuth 时禁用按钮且不 emit submit。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentComposer.oauthGate.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')

  const { mockApiFetch } = vi.hoisted(() => ({ mockApiFetch: vi.fn() }))

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: mockApiFetch,
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/t', query: {}, hash: '' }),
    useRouter: () => ({ replace: () => {} }),
  }))
  vi.mock('../../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => 'u1',
  }))

  const { default: TaskDetailCommentComposer } = await import('./TaskDetailCommentComposer.vue')
  const { rememberGrantTicket } = await import('../../utils/grantTicketSession.js')

  describe('TaskDetailCommentComposer OAuth gate', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      sessionStorage.clear()
      mockApiFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ container_image_at_mode_enabled: true }),
      })
    })

    const mountComposer = () => mount(TaskDetailCommentComposer, {
      props: {
        modelValue: 'please fix',
        'onUpdate:modelValue': () => {},
        tenantId: 't1',
        workspaceId: 'w1',
        showDependencyPicker: false,
        taskProjectsWithDetails: [
          { project: { git_repos: ['https://github.com/acme/demo.git'] } },
        ],
      },
      global: {
        stubs: {
          CommentImageMentionEditor: { template: '<div />' },
          CommentExecutionDependencyPicker: true,
          ServerConfigImageSectionHints: true,
          CommentComposerHardwareCard: true,
          CommentComposerRepoIdentity: {
            template: '<div data-testid="comment-composer-repo-identity" />',
          },
          ServerConfigFeatureParamsBlock: true,
        },
      },
    })

    it('plain comment stays enabled without oauth reason', async () => {
      const wrapper = mountComposer()
      await flushPromises()
      await nextTick()
      const btn = wrapper.get('[data-testid="task-detail-comment-submit"]')
      expect(btn.text()).toContain('提交评论')
      expect(btn.attributes('disabled')).toBeUndefined()
      expect(wrapper.find('[data-testid="comment-composer-oauth-blocked-reason"]').exists()).toBe(false)
      await btn.trigger('click')
      expect(wrapper.emitted('submit')).toHaveLength(1)
      wrapper.unmount()
    })

    it('disables submit-and-run when oauth is unbound and does not emit submit', async () => {
      const wrapper = mountComposer()
      await flushPromises()
      wrapper.vm.onMentionChange({ id: 'img-a', name: 'trae-agent' })
      wrapper.vm.onRepoOAuthReadiness({
        hasOAuthRepos: true,
        loading: false,
        allBound: false,
        checkError: '',
      })
      await nextTick()
      const btn = wrapper.get('[data-testid="task-detail-comment-submit"]')
      expect(btn.text()).toContain('提交并运行')
      expect(btn.attributes('disabled')).toBeDefined()
      expect(wrapper.get('[data-testid="comment-composer-oauth-blocked-reason"]').text())
        .toMatch(/提交并运行前/)
      await btn.trigger('click')
      expect(wrapper.emitted('submit')).toBeFalsy()
      wrapper.unmount()
    })

    it('emits submit-and-run when oauth is bound', async () => {
      rememberGrantTicket('tkt-1', 'https://github.com/acme/demo.git')
      const wrapper = mountComposer()
      await flushPromises()
      wrapper.vm.onMentionChange({ id: 'img-a', name: 'trae-agent' })
      wrapper.vm.onRepoOAuthReadiness({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
        checkError: '',
      })
      await nextTick()
      const btn = wrapper.get('[data-testid="task-detail-comment-submit"]')
      expect(btn.attributes('disabled')).toBeUndefined()
      await btn.trigger('click')
      expect(wrapper.emitted('submit')).toHaveLength(1)
      wrapper.unmount()
    })

    it('keeps submit-and-run disabled when L1 is bound but session grant ticket is missing', async () => {
      const wrapper = mountComposer()
      await flushPromises()
      wrapper.vm.onMentionChange({ id: 'img-a', name: 'trae-agent' })
      wrapper.vm.onRepoOAuthReadiness({
        hasOAuthRepos: true,
        loading: false,
        allBound: true,
        checkError: '',
      })
      await nextTick()
      const btn = wrapper.get('[data-testid="task-detail-comment-submit"]')
      expect(btn.attributes('disabled')).toBeDefined()
      expect(wrapper.get('[data-testid="comment-composer-oauth-blocked-reason"]').text())
        .toMatch(/使用授权/)
      await btn.trigger('click')
      expect(wrapper.emitted('submit')).toBeFalsy()
      wrapper.unmount()
    })
  })
}
