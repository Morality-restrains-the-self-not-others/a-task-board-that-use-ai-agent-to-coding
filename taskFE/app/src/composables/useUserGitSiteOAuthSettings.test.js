if (!process.env.VITEST) {
  console.log('[skip] useUserGitSiteOAuthSettings.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const { buildProviderAppStartUrl } = await import('./useUserGitSiteOAuthSettings.js')

describe('buildProviderAppStartUrl', () => {
  it('github → /api/git-oauth/github-app-start/ 且携带参数', () => {
    const params = new URLSearchParams({ next: '/projects/', return_key: 'rk1', service_provider: 'default' })
    expect(buildProviderAppStartUrl('github', params)).toBe(
      '/api/git-oauth/github-app-start/?next=%2Fprojects%2F&return_key=rk1&service_provider=default',
    )
  })

  it('gitlab → /api/git-oauth/gitlab-app-start/', () => {
    const params = new URLSearchParams({ next: '/user/u1/profile/git-site-oauth/' })
    expect(buildProviderAppStartUrl('gitlab', params)).toBe(
      '/api/git-oauth/gitlab-app-start/?next=%2Fuser%2Fu1%2Fprofile%2Fgit-site-oauth%2F',
    )
  })

  it('无参数时不带 ? 后缀', () => {
    expect(buildProviderAppStartUrl('github')).toBe('/api/git-oauth/github-app-start/')
  })

  it('不使用网关无路由的旧路径 /api/accounts/{provider}/app/start/（回归：P2）', () => {
    const url = buildProviderAppStartUrl('github', new URLSearchParams({ next: '/' }))
    expect(url).not.toMatch(/^\/api\/accounts\//)
    expect(url).toMatch(/^\/api\/git-oauth\/github-app-start\//)
  })
})

}
