// @vitest-environment node
/**
 * OPT-20260815-019：评论级克隆进度条生效后，任务级「容器项目克隆进度」横幅收起，
 * 避免同一克隆进度在评论列表顶端与评论「执行细节」两处展示。
 * 纯逻辑在 commentCloneProgressFromLogs.test.js（hasActiveCommentCloneProgress），
 * 此处用源码扫描守住 bindings 的接线不回归。
 */
if (!process.env.VITEST) {
  console.log('[skip] taskDetailSectionBindings.cloneBanner.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))

  describe('任务级克隆横幅受评论级进度门控', () => {
    it('bindings 用 hasActiveCommentCloneProgress 收起 showContainerCloneProgressBanner', () => {
      const bindings = readFileSync(join(here, 'taskDetailSectionBindings.js'), 'utf8')
      expect(bindings).toMatch(/hasActiveCommentCloneProgress/)
      expect(bindings).toMatch(/showContainerCloneProgressBanner:/)
      expect(bindings).toMatch(/!hasActiveCommentCloneProgress\(containerCloneProgressByCommentId\?\.value\)/)
    })
  })
}
