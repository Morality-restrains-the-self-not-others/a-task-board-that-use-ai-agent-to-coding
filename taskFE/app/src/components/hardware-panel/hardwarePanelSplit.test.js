// @vitest-environment node
/**
 * 硬件面板区域拆分回归：主壳 ≤500 行，关键 data-testid / 子组件挂载点保留。
 */
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(fileURLToPath(import.meta.url))
const shellPath = join(root, '..', 'ServerConfigHardwarePanel.vue')
const composablePath = join(
  root,
  '..',
  '..',
  'composables',
  'hardwarePanel',
  'useServerConfigHardwarePanel.js',
)

if (!process.env.VITEST) {
  console.log('[skip] hardwarePanelSplit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')

  describe('ServerConfigHardwarePanel 区域拆分', () => {
    it('主壳物理行数 ≤500 且引用 composable 与子组件', () => {
      const src = readFileSync(shellPath, 'utf8')
      const lines = src.split(/\r?\n/).length
      expect(lines).toBeLessThanOrEqual(500)
      expect(src).toContain('useServerConfigHardwarePanel')
      expect(src).toContain('data-testid="server-hardware-config-panel"')
      expect(src).toContain('HardwarePanelHeader')
      expect(src).toContain('HardwareNetworkSelectors')
      expect(src).toContain('HardwareInstanceListPanel')
      expect(src).not.toContain('HardwareServerControls')
      expect(src).not.toContain('start-server-btn')
      expect(src).not.toContain('start-server-disabled-reason')
    })

    it('composable 导出 useServerConfigHardwarePanel 且含启动相关 API', () => {
      const src = readFileSync(composablePath, 'utf8')
      expect(src).toContain('export function useServerConfigHardwarePanel')
      expect(src).toContain('startServer')
      expect(src).toContain('buildRunTemplatePayload')
      expect(src).toContain('hardwarePanelStartVm')
      // 拆分后若丢失这些定义，setup 会 ReferenceError，整块硬件面板（含自动释放）不挂载
      expect(src).toMatch(/const showProjectTemplateSummaryOnly\s*=\s*computed/)
      expect(src).toMatch(/const temporaryConfigExpanded\s*=\s*ref\(/)
      expect(src).toMatch(/const showHardwareConfigForm\s*=\s*computed/)
      expect(src).toMatch(/const panelExpanded\s*=\s*ref\(/)
    })
  })
}
