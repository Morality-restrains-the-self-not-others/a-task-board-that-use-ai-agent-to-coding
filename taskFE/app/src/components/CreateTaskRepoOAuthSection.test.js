// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] CreateTaskRepoOAuthSection.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: CreateTaskRepoOAuthSection } = await import('./CreateTaskRepoOAuthSection.vue')

  const gitlabUrl = 'https://gitlab-tencent-sh-1.daydaymoney.com/g/demo.git'

  describe('CreateTaskRepoOAuthSection', () => {
    it('shows bound copy from parent oauthRepoUrls when that url is token available', () => {
      const wrapper = mount(CreateTaskRepoOAuthSection, {
        props: {
          editingTask: { auto_run: true, projectSelections: [{ projectId: 'p1' }] },
          projects: [{ id: 'p1', git_repos: [gitlabUrl] }],
          oauthRepoUrls: [gitlabUrl],
          oauthBoundByUrl: { [gitlabUrl]: true },
          oauthLoadingByUrl: { [gitlabUrl]: false },
        },
      })
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bound"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="create-task-repo-oauth-bind"]').exists()).toBe(false)
      wrapper.unmount()
    })

    it('shows bind link when parent reports the same url unbound', () => {
      const wrapper = mount(CreateTaskRepoOAuthSection, {
        props: {
          editingTask: { auto_run: true, projectSelections: [{ projectId: 'p1' }] },
          projects: [{ id: 'p1', git_repos: [gitlabUrl] }],
          oauthRepoUrls: [gitlabUrl],
          oauthBoundByUrl: { [gitlabUrl]: false },
          oauthLoadingByUrl: { [gitlabUrl]: false },
        },
      })
      const bind = wrapper.find('[data-testid="create-task-repo-oauth-bind"]')
      expect(bind.exists()).toBe(true)
      expect(bind.element.tagName).toBe('A')
      expect(bind.attributes('href')).toContain('repo_url=')
      wrapper.unmount()
    })
  })
}
