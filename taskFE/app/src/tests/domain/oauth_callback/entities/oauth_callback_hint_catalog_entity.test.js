// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] oauth_callback_hint_catalog_entity.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { OAuthCallbackHintCatalog } = await import(
    '../../../../domain/oauth_callback/entities/oauth_callback_hint_catalog_entity.js'
  )

  describe('OAuthCallbackHintCatalog', () => {
    it('ok → success 语义', () => {
      const catalog = new OAuthCallbackHintCatalog()
      const out = catalog.resolveRawMessage('github', 'ok')
      expect(out.severity).toBe('success')
      expect(out.message).toContain('授权成功')
    })

    it('exchange_failed 映射网络语义文案（2026-08-07 优化）', () => {
      const catalog = new OAuthCallbackHintCatalog()
      expect(catalog.resolveRawMessage('github', 'exchange_failed').message).toBe(
        '授权失败：无法连接 GitHub 服务器，请检查网络后重试',
      )
      expect(catalog.resolveRawMessage('gitlab', 'exchange_failed').message).toBe(
        '授权失败：无法连接 GitLab 服务器，请检查网络后重试',
      )
    })

    it('exchange_rejected 映射「GitHub 拒绝」语义文案（2026-08-07 新码）', () => {
      const catalog = new OAuthCallbackHintCatalog()
      const out = catalog.resolveRawMessage('github', 'exchange_rejected')
      expect(out.severity).toBe('error')
      expect(out.message).toContain('GitHub 拒绝了授权请求')
    })

    it('exchange_rejected 映射「GitLab 拒绝」语义文案（2026-08-08 对齐 GitLab）', () => {
      const catalog = new OAuthCallbackHintCatalog()
      const out = catalog.resolveRawMessage('gitlab', 'exchange_rejected')
      expect(out.severity).toBe('error')
      expect(out.message).toContain('GitLab 拒绝了授权请求')
    })

    it('未知码回退「授权未完成（code）」', () => {
      const catalog = new OAuthCallbackHintCatalog()
      expect(catalog.resolveRawMessage('github', 'weird_code').message).toBe(
        '授权未完成（weird_code）',
      )
    })
  })
}
