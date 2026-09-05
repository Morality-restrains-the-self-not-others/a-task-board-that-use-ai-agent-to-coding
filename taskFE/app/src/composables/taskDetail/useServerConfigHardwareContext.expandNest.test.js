// @vitest-environment jsdom
/**
 * expandHardwareForComment：硬件迁入镜像卡后滚动定位，不再切换 Tab。
 */
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigHardwareContext.expandNest.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
  const { expandHardwareForComment } = await import('./useServerConfigHardwareContext.js')

  describe('expandHardwareForComment 镜像卡内定位', () => {
    let scrollIntoView

    beforeEach(() => {
      scrollIntoView = vi.fn()
      const el = document.createElement('div')
      el.id = 'hardware-config-section'
      el.scrollIntoView = scrollIntoView
      document.body.appendChild(el)
    })

    afterEach(() => {
      document.body.innerHTML = ''
    })

    it('打开临时配置并 scrollIntoView，不改 activeServerSection', () => {
      const expandServerSectionBody = vi.fn()
      const activeServerSection = { value: 'runtime' }
      const openTemporaryConfig = vi.fn()
      const serverHardwarePanelRef = { value: { openTemporaryConfig } }

      expandHardwareForComment(expandServerSectionBody, activeServerSection, serverHardwarePanelRef)

      expect(expandServerSectionBody).toHaveBeenCalled()
      expect(openTemporaryConfig).toHaveBeenCalled()
      expect(scrollIntoView).toHaveBeenCalled()
      expect(activeServerSection.value).toBe('runtime')
    })

    it('源码不再 provide serverConfigHardwareContext', async () => {
      const { readFileSync } = await import('node:fs')
      const { dirname, join } = await import('node:path')
      const { fileURLToPath } = await import('node:url')
      const src = readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'useServerConfigHardwareContext.js'), 'utf8')
      expect(src).not.toContain("provide('serverConfigHardwareContext'")
      expect(src).not.toMatch(/export function useServerConfigHardwareContext/)
    })
  })
}
