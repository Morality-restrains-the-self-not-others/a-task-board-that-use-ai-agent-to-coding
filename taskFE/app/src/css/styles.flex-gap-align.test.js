if (!process.env.VITEST) {
  console.log('[skip] styles.flex-gap-align.test.js requires vitest runtime')
} else {
  /**
   * 回归：禁止用 .flex.gap-3 高特异性强制 align-items:center。
   * 该规则会盖过 Tailwind items-start，把评论头像垂直居中到整卡中部。
   */
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))

  describe('styles.css flex gap alignment', () => {
    it('does not force align-items:center on .flex.gap-3 with class-level specificity', () => {
      const css = readFileSync(join(here, 'styles.css'), 'utf8')
      expect(css).not.toMatch(/\.flex\.gap-3\s*,\s*\.flex\.space-x-3\s*\{[^}]*align-items:\s*center/)
    })
  })
}
