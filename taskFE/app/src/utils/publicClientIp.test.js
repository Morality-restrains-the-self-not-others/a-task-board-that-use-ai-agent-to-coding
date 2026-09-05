// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] publicClientIp.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')

  describe('fetchPublicClientIp', () => {
    beforeEach(() => {
      vi.resetModules()
    })

    afterEach(() => {
      vi.clearAllMocks()
    })

    it('返回 API 的 ip 并缓存第二次调用不再请求', async () => {
      const apiFetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({ ip: '203.0.113.44' }),
      })
      vi.doMock('./apiUtils.js', () => ({ apiFetch }))

      const mod = await import('./publicClientIp.js')
      mod.__resetPublicClientIpCacheForTests()
      const a = await mod.fetchPublicClientIp()
      const b = await mod.fetchPublicClientIp()
      expect(a).toBe('203.0.113.44')
      expect(b).toBe('203.0.113.44')
      expect(apiFetch).toHaveBeenCalledTimes(1)
      expect(String(apiFetch.mock.calls[0][0])).toContain('/api/accounts/users/client-ip/')
    })

    it('force=true 时忽略缓存再次请求', async () => {
      const apiFetch = vi.fn()
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({ ip: '203.0.113.1' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => ({ ip: '203.0.113.2' }),
        })
      vi.doMock('./apiUtils.js', () => ({ apiFetch }))

      const mod = await import('./publicClientIp.js')
      mod.__resetPublicClientIpCacheForTests()
      const a = await mod.fetchPublicClientIp()
      const b = await mod.fetchPublicClientIp({ force: true })
      expect(a).toBe('203.0.113.1')
      expect(b).toBe('203.0.113.2')
      expect(apiFetch).toHaveBeenCalledTimes(2)
    })

    it('HTTP 非 2xx 时抛错', async () => {
      const apiFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        json: async () => ({ detail: 'fail' }),
      })
      vi.doMock('./apiUtils.js', () => ({ apiFetch }))

      const mod = await import('./publicClientIp.js')
      mod.__resetPublicClientIpCacheForTests()
      await expect(mod.fetchPublicClientIp()).rejects.toThrow(/client-ip http 502/)
    })
  })
}
