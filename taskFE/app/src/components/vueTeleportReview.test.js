// @vitest-environment node
/**
 * OPT-20260813-015：功能卡就地挂载；存量 Teleport 须有 Teleport-OK。
 */
if (!process.env.VITEST) {
  console.log('[skip] vueTeleportReview.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { execSync } = await import('node:child_process')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  const here = dirname(fileURLToPath(import.meta.url))
  const taskFeRoot = join(here, '../../..')

  describe('taskFE Teleport 重审', () => {
    it('ServerConfig.logic 不再 Teleport 功能卡；composer 就地挂载 FeatureParamsBlock', () => {
      const logicSrc = readFileSync(join(here, 'ServerConfig.logic.vue'), 'utf8')
      const composerSrc = readFileSync(
        join(here, 'task-detail', 'TaskDetailCommentComposer.vue'),
        'utf8',
      )
      expect(logicSrc).not.toContain('<Teleport')
      expect(logicSrc).not.toContain('comment-composer-feature-params-slot')
      expect(logicSrc).not.toContain('ServerConfigFeatureParamsBlock')
      expect(composerSrc).toContain('ServerConfigFeatureParamsBlock')
      const slotIdx = composerSrc.indexOf('comment-composer-feature-params-slot')
      const blockIdx = composerSrc.indexOf('<ServerConfigFeatureParamsBlock')
      const hwIdx = composerSrc.indexOf('<CommentComposerHardwareCard')
      expect(slotIdx).toBeGreaterThan(-1)
      expect(blockIdx).toBeGreaterThan(slotIdx)
      expect(hwIdx).toBeGreaterThan(blockIdx)
    })

    it('存量 Teleport 均带 Teleport-OK 注释，且无 data-testid 槽传送', () => {
      const out = execSync(
        `rg -n --glob '*.vue' '<Teleport' ${taskFeRoot}`,
        { encoding: 'utf8' },
      )
      const lines = out.trim().split('\n').filter(Boolean)
      expect(lines.length).toBeGreaterThan(0)
      for (const line of lines) {
        const [file, lineNo] = line.split(':')
        const src = readFileSync(file, 'utf8')
        const n = Number(lineNo)
        const nearby = src.split('\n').slice(Math.max(0, n - 4), n).join('\n')
        expect(nearby, line).toMatch(/Teleport-OK:/)
        expect(src).not.toMatch(/<Teleport[^>]*to="\[data-testid/)
      }
    })
  })
}
