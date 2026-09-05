// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentsPanel.noCommentRuntimeForwarding.test.js requires vitest runtime')
} else {
  /**
   * OPT-20260815-017：评论区硬件内联面板的 start-request-accepted / stop-server
   * 转发已被移除（停机/启动入口只留在评论「执行细节」运行态 Tab）。
   * 用源码扫描守住：CommentsPanel 不再声明这两类事件，CommentsSection 不再向 Panel 监听并上抛。
   */
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))

  describe('TaskDetailCommentsPanel 不再转发评论级运行态操作', () => {
    it('Panel 与 Section 源码均不再出现 start-request-accepted / stop-server', () => {
      const panel = readFileSync(join(here, 'TaskDetailCommentsPanel.vue'), 'utf8')
      const section = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')

      expect(panel, 'panel').not.toMatch(/start-request-accepted/)
      expect(panel, 'panel').not.toMatch(/stop-server/)
      expect(section, 'section').not.toMatch(/start-request-accepted/)
      expect(section, 'section').not.toMatch(/stop-server/)
    })
  })
}
