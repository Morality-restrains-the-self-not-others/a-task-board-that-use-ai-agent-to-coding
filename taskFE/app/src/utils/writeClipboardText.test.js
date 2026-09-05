// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] writeClipboardText.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { writeClipboardText } = await import('./writeClipboardText.js')

  describe('writeClipboardText', () => {
    it('uses navigator.clipboard.writeText when available', async () => {
      const writeText = vi.fn(async () => {})
      vi.stubGlobal('navigator', { clipboard: { writeText } })
      await writeClipboardText('hello')
      expect(writeText).toHaveBeenCalledWith('hello')
      vi.unstubAllGlobals()
    })

    it('no-ops on empty text', async () => {
      const writeText = vi.fn(async () => {})
      vi.stubGlobal('navigator', { clipboard: { writeText } })
      await writeClipboardText('')
      expect(writeText).not.toHaveBeenCalled()
      vi.unstubAllGlobals()
    })
  })
}
