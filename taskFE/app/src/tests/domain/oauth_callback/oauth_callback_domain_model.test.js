// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] oauth_callback_domain_model.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { OAuthGitProvider } = await import(
    '../../../domain/oauth_callback/value_objects/oauth_git_provider_value_object.js'
  )
  const { OAuthCallbackRouteSnapshot } = await import(
    '../../../domain/oauth_callback/entities/oauth_callback_route_snapshot_entity.js'
  )
  const { InMemoryOAuthCallbackHintCatalogRepository } = await import(
    '../../../domain/oauth_callback/repositories/oauth_callback_hint_catalog_in_memory_repository.js'
  )
  const { InMemoryOAuthCallbackToastSkipPolicyRepository } = await import(
    '../../../domain/oauth_callback/repositories/oauth_callback_toast_skip_policy_in_memory_repository.js'
  )
  const { OAuthCallbackMessageResolutionService } = await import(
    '../../../domain/oauth_callback/services/oauth_callback_message_resolution_service.js'
  )
  const { OAuthCallbackRouteApplicationService } = await import(
    '../../../domain/oauth_callback/services/oauth_callback_route_application_service.js'
  )

  describe('OAuthGitProvider', () => {
    it('拒绝未知 provider', () => {
      expect(() => new OAuthGitProvider('bitbucket')).toThrow(/无效的 OAuth provider/)
    })
  })

  describe('OAuthCallbackRouteSnapshot', () => {
    it('gitlab query 优先于 github', () => {
      const snapshot = new OAuthCallbackRouteSnapshot({
        path: '/tenant/1/projects/2/',
        query: { gitlab: 'profile_failed', github: 'ok' },
      })
      const detected = snapshot.detectCallbackParam()
      expect(detected.provider.value).toBe('gitlab')
      expect(detected.code.value).toBe('profile_failed')
    })

    it('清除 query 键后保留其它参数并去掉 trace_id', () => {
      const snapshot = new OAuthCallbackRouteSnapshot({
        path: '/p/',
        query: { gitlab: 'bad_state', foo: 'bar', trace_id: 'tid-x' },
      })
      expect(snapshot.queryWithoutKey('gitlab')).toEqual({ foo: 'bar' })
      expect(snapshot.callbackTraceId()).toBe('tid-x')
    })
  })

  describe('OAuthCallbackMessageResolutionService', () => {
    it('profile_failed 映射 GitLab 文案', () => {
      const service = new OAuthCallbackMessageResolutionService(
        new InMemoryOAuthCallbackHintCatalogRepository(),
      )
      const outcome = service.resolve('gitlab', 'profile_failed')
      expect(outcome.severityValue).toBe('error')
      expect(outcome.messageValue).toContain('无法读取 GitLab 用户资料')
    })

    it('exchange_failed 映射网络语义文案（2026-08-07 优化）', () => {
      const service = new OAuthCallbackMessageResolutionService(
        new InMemoryOAuthCallbackHintCatalogRepository(),
      )
      const gh = service.resolve('github', 'exchange_failed')
      expect(gh.messageValue).toBe('授权失败：无法连接 GitHub 服务器，请检查网络后重试')
      const gl = service.resolve('gitlab', 'exchange_failed')
      expect(gl.messageValue).toBe('授权失败：无法连接 GitLab 服务器，请检查网络后重试')
    })

    it('exchange_rejected 映射「GitHub 拒绝」语义文案（2026-08-07 新码）', () => {
      const service = new OAuthCallbackMessageResolutionService(
        new InMemoryOAuthCallbackHintCatalogRepository(),
      )
      const outcome = service.resolve('github', 'exchange_rejected')
      expect(outcome.severityValue).toBe('error')
      expect(outcome.messageValue).toContain('GitHub 拒绝了授权请求')
    })
  })

  describe('OAuthCallbackRouteApplicationService', () => {
    it('无回调 query 时返回 null', () => {
      const service = new OAuthCallbackRouteApplicationService(
        new OAuthCallbackMessageResolutionService(
          new InMemoryOAuthCallbackHintCatalogRepository(),
        ),
      )
      expect(service.consumeFromRoute({ path: '/p/', query: {} })).toBeNull()
    })
  })

  describe('InMemoryOAuthCallbackToastSkipPolicyRepository', () => {
    it('git-site-oauth 路径应 skip', () => {
      const repo = new InMemoryOAuthCallbackToastSkipPolicyRepository()
      expect(repo.shouldSkipToast('/user/1/profile/git-site-oauth/')).toBe(true)
      expect(repo.shouldSkipToast('/tenant/1/projects/2/')).toBe(false)
    })
  })
}
