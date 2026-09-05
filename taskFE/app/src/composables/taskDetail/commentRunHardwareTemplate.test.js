// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] commentRunHardwareTemplate.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    resolveCommentServerRunTemplate,
    resolveCommentHardwareRunHint,
    registerCommentHardwarePanelReader,
    readRegisteredCommentHardwarePanel,
  } = await import('./commentRunHardwareTemplate.js')

  describe('resolveCommentServerRunTemplate', () => {
    it('returns null for project template source', () => {
      expect(resolveCommentServerRunTemplate({
        getHardwareConfigSource: () => 'project_template',
        buildRunTemplatePayload: () => ({ cloud_platform_id: 'p', region: 'cn-hangzhou' }),
      })).toBeNull()
    })

    it('returns payload when temporary config is complete', () => {
      const payload = { cloud_platform_id: 'plat-1', region: 'cn-hangzhou', zone_id: 'b' }
      expect(resolveCommentServerRunTemplate({
        hardwareConfigSource: 'temporary',
        buildRunTemplatePayload: () => payload,
      })).toEqual(payload)
    })

    it('returns null when temporary payload missing region', () => {
      expect(resolveCommentServerRunTemplate({
        hardwareConfigSource: 'temporary',
        buildRunTemplatePayload: () => ({ cloud_platform_id: 'plat-1' }),
      })).toBeNull()
    })
  })

  describe('resolveCommentHardwareRunHint', () => {
    it('warns incomplete temporary config cannot run but does not treat as submit blocker copy', () => {
      const hint = resolveCommentHardwareRunHint({
        getHardwareConfigSource: () => 'temporary',
        buildRunTemplatePayload: () => ({}),
      })
      expect(hint).toMatch(/无法启动运行/)
      expect(hint).toMatch(/仍可发送评论/)
      expect(hint).not.toMatch(/后再提交/)
    })

    it('has no hint in project template mode', () => {
      expect(resolveCommentHardwareRunHint({
        getHardwareConfigSource: () => 'project_template',
        buildRunTemplatePayload: () => ({}),
      })).toBe('')
    })
  })

  describe('registerCommentHardwarePanelReader', () => {
    it('stores and clears the panel reader', () => {
      const panel = { hardwareConfigSource: 'temporary' }
      registerCommentHardwarePanelReader(() => panel)
      expect(readRegisteredCommentHardwarePanel()).toBe(panel)
      registerCommentHardwarePanelReader(() => null)
      expect(readRegisteredCommentHardwarePanel()).toBeNull()
    })
  })
}
