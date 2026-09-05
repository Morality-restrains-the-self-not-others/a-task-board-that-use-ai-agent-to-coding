// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] githubAppContinueNav.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    buildOauthReturnPath,
    mergeQueryParam,
    resolveGithubAppContinueHref,
    hrefToRouterLocation,
  } = await import('./githubAppContinueNav.js')

  const PROJECT_RETURN =
    '/tenant/877397588196749312/projects/proj_880498883115905024/?accessCode=9aaHjbryhL'

  describe('buildOauthReturnPath', () => {
    it('prefers location.search including accessCode', () => {
      expect(
        buildOauthReturnPath({
          routePath: '/tenant/t/projects/p/',
          pathname: '/tenant/t/projects/p/',
          search: '?accessCode=9aaHjbryhL',
          query: {},
        }),
      ).toBe('/tenant/t/projects/p/?accessCode=9aaHjbryhL')
    })

    it('falls back to route.query when location.search is empty', () => {
      expect(
        buildOauthReturnPath({
          routePath: '/tenant/t/projects/p/',
          pathname: '/tenant/t/projects/p/',
          search: '',
          query: { accessCode: '9aaHjbryhL' },
        }),
      ).toBe('/tenant/t/projects/p/?accessCode=9aaHjbryhL')
    })
  })

  describe('mergeQueryParam', () => {
    it('appends gitlab=ok without dropping accessCode', () => {
      expect(mergeQueryParam(PROJECT_RETURN, 'gitlab', 'ok')).toBe(
        `${PROJECT_RETURN}&gitlab=ok`,
      )
    })

    it('does not duplicate an existing key', () => {
      expect(mergeQueryParam(`${PROJECT_RETURN}&gitlab=ok`, 'gitlab', 'ok')).toBe(
        `${PROJECT_RETURN}&gitlab=ok`,
      )
    })
  })

  describe('resolveGithubAppContinueHref', () => {
    it('returns project page with accessCode and gitlab=ok after retry', () => {
      const href = resolveGithubAppContinueHref({
        returnUrl: PROJECT_RETURN,
        provider: 'gitlab',
        providerParam: 'ok',
        traceId: '',
      })
      expect(href).toContain('/projects/proj_880498883115905024/')
      expect(href).toContain('accessCode=9aaHjbryhL')
      expect(href).toContain('gitlab=ok')
      expect(href).not.toContain('/profile/git-site-oauth')
    })

    it('attaches trace_id only on non-ok provider param', () => {
      const href = resolveGithubAppContinueHref({
        returnUrl: PROJECT_RETURN,
        provider: 'gitlab',
        providerParam: 'profile_failed',
        traceId: 'tid-1',
      })
      expect(href).toContain('trace_id=tid-1')
      expect(href).toContain('accessCode=9aaHjbryhL')
    })
  })

  describe('hrefToRouterLocation', () => {
    it('splits path and query so Vue Router keeps accessCode', () => {
      const loc = hrefToRouterLocation(`${PROJECT_RETURN}&gitlab=ok`)
      expect(loc.path).toBe('/tenant/877397588196749312/projects/proj_880498883115905024/')
      expect(loc.query.accessCode).toBe('9aaHjbryhL')
      expect(loc.query.gitlab).toBe('ok')
    })

    it('returns path-only when there is no query', () => {
      expect(hrefToRouterLocation('/profile/')).toEqual({ path: '/profile/', query: {} })
    })
  })
}
