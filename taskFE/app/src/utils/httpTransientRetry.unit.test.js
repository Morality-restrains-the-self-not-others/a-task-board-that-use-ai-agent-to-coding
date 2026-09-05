// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] httpTransientRetry.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { retryTransientHttp, transientHttpRetry } = await import('./httpTransientRetry.js')

  beforeEach(() => {
    transientHttpRetry.sleep = async () => {}
    transientHttpRetry.delaysMs = [0, 0, 0]
    transientHttpRetry.extraAttempts = 3
  })

  describe('retryTransientHttp', () => {
    it('returns the first ok response without extra calls', async () => {
      const fn = vi.fn().mockResolvedValue({ ok: true, status: 200 })
      const resp = await retryTransientHttp(fn, { url: '/ok' })
      expect(resp.status).toBe(200)
      expect(fn).toHaveBeenCalledTimes(1)
    })

    it('retries 502 then returns the later 200', async () => {
      const fn = vi.fn()
        .mockResolvedValueOnce({ ok: false, status: 502, traceId: 't-1' })
        .mockResolvedValueOnce({ ok: true, status: 200 })
      const onRetry = vi.fn()
      const resp = await retryTransientHttp(fn, { url: '/feature-params', onRetry })
      expect(resp.ok).toBe(true)
      expect(fn).toHaveBeenCalledTimes(2)
      expect(onRetry).toHaveBeenCalledWith(expect.objectContaining({
        attempt: 1,
        status: 502,
        url: '/feature-params',
        traceId: 't-1',
      }))
    })

    it('retries fetch throws then returns ok', async () => {
      const fn = vi.fn()
        .mockRejectedValueOnce(new TypeError('Failed to fetch'))
        .mockResolvedValueOnce({ ok: true, status: 200 })
      const resp = await retryTransientHttp(fn)
      expect(resp.ok).toBe(true)
      expect(fn).toHaveBeenCalledTimes(2)
    })

    it('exhausts retries and returns the last 502', async () => {
      const fn = vi.fn().mockResolvedValue({ ok: false, status: 502, traceId: 't-ex' })
      const resp = await retryTransientHttp(fn)
      expect(resp.status).toBe(502)
      expect(fn).toHaveBeenCalledTimes(4)
    })

    it('does not retry business 403', async () => {
      const fn = vi.fn().mockResolvedValue({ ok: false, status: 403 })
      const resp = await retryTransientHttp(fn)
      expect(resp.status).toBe(403)
      expect(fn).toHaveBeenCalledTimes(1)
    })

    it('rethrows the last network error after exhaustion', async () => {
      const fn = vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
      await expect(retryTransientHttp(fn)).rejects.toThrow('Failed to fetch')
      expect(fn).toHaveBeenCalledTimes(4)
    })
  })
}
