// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailImageSelectionBridge.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { ref } = await import('vue')
  const {
    bridgedSelectedImageId,
    bindServerConfigSelectedImage,
  } = await import('./taskDetailImageSelectionBridge.js')

  describe('taskDetailImageSelectionBridge', () => {
    it('双向同步 selectedImageId 与 bridge', () => {
      bridgedSelectedImageId.value = ''
      const selectedImageId = ref('img-1')
      const stop = bindServerConfigSelectedImage(selectedImageId)
      expect(bridgedSelectedImageId.value).toBe('img-1')

      bridgedSelectedImageId.value = 'img-2'
      expect(selectedImageId.value).toBe('img-2')

      selectedImageId.value = 'img-3'
      expect(bridgedSelectedImageId.value).toBe('img-3')
      stop()
    })
  })
}
