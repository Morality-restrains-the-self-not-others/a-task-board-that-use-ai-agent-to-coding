// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserProfileWechatBindingPanel.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({ apiFetch }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言解绑 DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-unbind-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('../domain/auth/services/saved_accounts_store.js', () => ({
    getActiveToken: async () => 'token-test',
  }))

  const { default: Panel } = await import('./UserProfileWechatBindingPanel.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('UserProfileWechatBindingPanel 解绑实施 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('点解绑发起一次 DELETE 且携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async () => jsonOk({ ok: true }))
      const wrapper = mount(Panel, {
        props: { hasWechat: true, wechatApps: ['web'], bindAvailable: true },
      })

      const unbindBtn = wrapper.findAll('button').find((b) => b.text().includes('解绑微信'))
      expect(unbindBtn).toBeTruthy()
      await unbindBtn.trigger('click')
      await flushPromises()

      const delCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/auth/wechat/unbind/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-unbind-1')
      expect(delCall[1].body).toContain('"app_key":"web"')
      wrapper.unmount()
    })

    it('解绑成功后发出 wechat-updated', async () => {
      apiFetch.mockImplementation(async () => jsonOk({ ok: true }))
      const wrapper = mount(Panel, {
        props: { hasWechat: true, wechatApps: ['web'], bindAvailable: true },
      })
      const unbindBtn = wrapper.findAll('button').find((b) => b.text().includes('解绑微信'))
      await unbindBtn.trigger('click')
      await flushPromises()
      expect(wrapper.emitted('wechat-updated')).toBeTruthy()
      wrapper.unmount()
    })
  })
}
