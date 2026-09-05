// @vitest-environment node
/**
 * 硬件摘要条已迁入 CommentComposerHardwareCard，评论区 bindings 不得再透传条级 props。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailSectionBindings.hardwareBar.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))

  describe('评论区硬件条死透传清理', () => {
    it('bindings / Section / Panel / TaskDetail 不再透传 showHardwareConfigBar 三项与 adjust/restore 转发', () => {
      const bindings = readFileSync(join(here, 'taskDetailSectionBindings.js'), 'utf8')
      const section = readFileSync(
        join(here, '../components/task-detail/TaskDetailCommentsSection.vue'),
        'utf8',
      )
      const panel = readFileSync(
        join(here, '../components/task-detail/TaskDetailCommentsPanel.vue'),
        'utf8',
      )
      const detail = readFileSync(join(here, 'TaskDetail.vue'), 'utf8')

      for (const [name, src] of [
        ['bindings', bindings],
        ['section', section],
        ['panel', panel],
      ]) {
        expect(src, name).not.toMatch(/showHardwareConfigBar/)
        expect(src, name).not.toMatch(/isUsingProjectHardwareTemplate/)
        expect(src, name).not.toMatch(/tempHardwareConfigLabel/)
      }

      expect(section).not.toMatch(/adjust-hardware-config/)
      expect(section).not.toMatch(/restore-hardware-defaults/)
      expect(panel).not.toMatch(/adjust-hardware-config/)
      expect(panel).not.toMatch(/restore-hardware-defaults/)
      expect(detail).not.toMatch(/handleAdjustHardwareConfig/)
      expect(detail).not.toMatch(/handleRestoreHardwareDefaults/)
      expect(detail).not.toMatch(/@adjust-hardware-config/)
      expect(detail).not.toMatch(/@restore-hardware-defaults/)
    })
  })
}
