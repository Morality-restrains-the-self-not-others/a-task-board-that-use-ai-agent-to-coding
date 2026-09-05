// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsGitlabOauthAppHelp.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: Help } = await import('./WorkspaceSettingsGitlabOauthAppHelp.vue')

  describe('WorkspaceSettingsGitlabOauthAppHelp', () => {
    it('documents Path A scopes and warns openid is not Path A', () => {
      const wrapper = mount(Help, {
        props: { redirectUri: 'https://www.daydaymoney.com/api/accounts/tenant-1/oauth/callback/' },
      })
      const text = wrapper.text()
      expect(text).toContain('write_repository')
      expect(text).toContain('openid')
      expect(text).toContain('Git 网站授权')
      expect(wrapper.find('[data-testid="gitlab-oauth-app-help"]').exists()).toBe(true)
    })
  })
}
