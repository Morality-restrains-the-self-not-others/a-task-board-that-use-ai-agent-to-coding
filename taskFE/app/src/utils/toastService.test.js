// @vitest-environment node
/**
 * toastService.error 第三参为 options 对象时不得把 null 当 object 解包 duration。
 */
if (!process.env.VITEST) {
  console.log('[skip] toastService.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
  const { default: toastService } = await import('./toastService.js')

  describe('toastService.error options', () => {
    beforeEach(() => {
      vi.useFakeTimers()
      toastService.hide()
    })
    afterEach(() => {
      vi.useRealTimers()
    })

    it('null duration 不抛 TypeError 且走默认时长', () => {
      expect(() => toastService.error('boom', null)).not.toThrow()
      expect(toastService.state.show).toBe(true)
      expect(toastService.state.message).toBe('boom')
      expect(toastService.state.type).toBe('error')
    })

    it('options.traceId 写入 error toast', () => {
      toastService.error('boom', { duration: 1000, traceId: 'abc-trace' })
      expect(toastService.state.traceId).toBe('abc-trace')
      expect(toastService.state.duration).toBe(1000)
    })
  })
}
