// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserCompanySettings.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PATCH 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-user' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 'tenant-1' } }),
  }))

  const { default: View } = await import('./UserCompanySettings.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('UserCompanySettings 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk({ personal_nickname: 'alice', company_nicknames: [] })
        }
        return jsonOk({ personal_nickname: 'alice', company_nicknames: [] })
      })
    })

    it('保存公司设置 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div />' },
            UserProfileCompanySettingsPanel: { template: '<div />' },
          },
        },
      })
      await flushPromises()

      const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存'))
      expect(saveBtn).toBeTruthy()
      await saveBtn.trigger('click')
      await flushPromises()

      const patchCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/accounts/users/profile/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-test-user')
    })
  })
}
