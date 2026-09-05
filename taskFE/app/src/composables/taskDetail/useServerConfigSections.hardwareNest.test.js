// @vitest-environment node
/**
 * 硬件迁入镜像区后：默认 Tab 不再指向 hardware。
 */
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigSections.hardwareNest.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const {
    useServerConfigSections,
    createServerConfigSectionHandlers,
  } = await import('./useServerConfigSections.js')

  describe('useServerConfigSections 硬件迁入镜像区', () => {
    it('非 relay 默认 activeServerSection 为 runtime（非 hardware）', () => {
      const route = { query: {} }
      const isRelayToTraeEnabled = { value: false }
      const { activeServerSection, applyDefaultServerTabByRuntime } = useServerConfigSections({
        route,
        isRelayToTraeEnabled,
      })
      expect(activeServerSection.value).toBe('runtime')
      expect(applyDefaultServerTabByRuntime(false)).toBe('runtime')
      expect(applyDefaultServerTabByRuntime(true)).toBe('runtime')
    })

    it('handlers 不再导出 onHardwareTabClick', () => {
      const activeServerSection = { value: 'runtime' }
      const handlers = createServerConfigSectionHandlers({
        activeServerSection,
        expandServerSectionBody: () => {},
        fetchServerContent: () => {},
        fetchServerStartHistory: () => {},
      })
      expect(handlers.onHardwareTabClick).toBeUndefined()
      expect(typeof handlers.selectServerSection).toBe('function')
    })
  })
}
