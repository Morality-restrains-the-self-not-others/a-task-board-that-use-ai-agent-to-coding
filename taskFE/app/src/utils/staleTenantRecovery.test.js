// @vitest-environment jsdom
/**
 * 陈旧租户恢复：清库/删公司后 URL 或 localStorage 仍指向不存在的 company id。
 */
if (!process.env.VITEST) {
  console.log('[skip] staleTenantRecovery.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it } = await import('vitest')
  const {
    LAST_TENANT_STORAGE_KEY,
    clearStaleLastActiveTenantId,
    resolveMissingCompanyRedirectPath,
    sanitizeStaleTenantPathSuffix,
  } = await import('./staleTenantRecovery.js')

  describe('clearStaleLastActiveTenantId', () => {
    beforeEach(() => {
      localStorage.clear()
    })

    it('匹配时清除 lastActiveTenantId', () => {
      localStorage.setItem(LAST_TENANT_STORAGE_KEY, '874599492341493760')
      expect(clearStaleLastActiveTenantId('874599492341493760')).toBe(true)
      expect(localStorage.getItem(LAST_TENANT_STORAGE_KEY)).toBeNull()
    })

    it('不匹配时保留原值', () => {
      localStorage.setItem(LAST_TENANT_STORAGE_KEY, '111')
      expect(clearStaleLastActiveTenantId('222')).toBe(false)
      expect(localStorage.getItem(LAST_TENANT_STORAGE_KEY)).toBe('111')
    })

    it('空 tenantId 或未设置时为 no-op', () => {
      expect(clearStaleLastActiveTenantId('')).toBe(false)
      expect(clearStaleLastActiveTenantId(null)).toBe(false)
      localStorage.setItem(LAST_TENANT_STORAGE_KEY, '111')
      expect(clearStaleLastActiveTenantId('   ')).toBe(false)
      expect(localStorage.getItem(LAST_TENANT_STORAGE_KEY)).toBe('111')
    })
  })

  describe('resolveMissingCompanyRedirectPath', () => {
    it('无公司 → /onboarding/', () => {
      expect(resolveMissingCompanyRedirectPath({ companies: [] })).toBe('/onboarding/')
      expect(resolveMissingCompanyRedirectPath({ companies: null })).toBe('/onboarding/')
      expect(resolveMissingCompanyRedirectPath({})).toBe('/onboarding/')
    })

    it('有公司时默认落到首个公司工作面板', () => {
      expect(
        resolveMissingCompanyRedirectPath({
          companies: [{ id: 'c1', name: 'A' }, { id: 'c2' }],
        }),
      ).toBe('/tenant/c1/work-panel/')
    })

    it('targetPathSuffix 可落到公司设置页', () => {
      expect(
        resolveMissingCompanyRedirectPath({
          companies: [{ id: '874600000000000001' }],
          targetPathSuffix: '/settings/company/',
        }),
      ).toBe('/tenant/874600000000000001/settings/company/')
    })

    it('跳过无效 id，选用下一个有效公司', () => {
      expect(
        resolveMissingCompanyRedirectPath({
          companies: [{ id: '' }, { id: '  ' }, { id: 'ok-id' }],
        }),
      ).toBe('/tenant/ok-id/work-panel/')
    })

    it('项目详情/编辑路径落到项目列表，不拷贝他公司 proj id', () => {
      expect(
        resolveMissingCompanyRedirectPath({
          companies: [{ id: 'c1' }],
          targetPathSuffix: '/projects/proj_880498883115905024/',
        }),
      ).toBe('/tenant/c1/projects/')
      expect(
        resolveMissingCompanyRedirectPath({
          companies: [{ id: 'c1' }],
          targetPathSuffix: '/projects/proj_880498883115905024/edit/',
        }),
      ).toBe('/tenant/c1/projects/')
    })

    it('任务详情与 workspace 资源路径落到工作面板', () => {
      expect(
        resolveMissingCompanyRedirectPath({
          companies: [{ id: 'c1' }],
          targetPathSuffix: '/workspace/ws_1/task-detail/9/',
        }),
      ).toBe('/tenant/c1/work-panel/')
    })
  })

  describe('sanitizeStaleTenantPathSuffix', () => {
    it('保留租户级列表/设置路径', () => {
      expect(sanitizeStaleTenantPathSuffix('/billing/orders/')).toBe('/billing/orders/')
      expect(sanitizeStaleTenantPathSuffix('/settings/company/')).toBe('/settings/company/')
      expect(sanitizeStaleTenantPathSuffix('/projects/')).toBe('/projects/')
    })

    it('剥掉项目与任务资源 id', () => {
      expect(sanitizeStaleTenantPathSuffix('/projects/proj_880498883115905024/')).toBe('/projects/')
      expect(sanitizeStaleTenantPathSuffix('/workspace/abc/task-detail/1/')).toBe('/work-panel/')
    })
  })
}
