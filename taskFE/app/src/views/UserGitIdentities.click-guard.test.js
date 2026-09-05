// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] UserGitIdentities.click-guard.test.js requires vitest runtime')
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
  // 断言 POST/PATCH 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-1' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { id: 'user-1' } }),
  }))
  vi.mock('../utils/sessionUserIdUtils.js', () => ({
    resolveAuthenticatedUserId: async () => 'user-1',
  }))

  const { default: View } = await import('./UserGitIdentities.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  describe('UserGitIdentities 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          if (String(url).includes('/profile/')) {
            return jsonOk({ company_nicknames: [{ company_id: 'comp-1', company_name: '测试公司' }] })
          }
          return jsonOk({ identities: [] })
        }
        return jsonOk({ ok: true })
      })
    })

    it('新建 Git 身份 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(View, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div />' },
            'router-link': { template: '<a><slot /></a>' },
            'router-view': true,
          },
        },
      })
      await flushPromises()
      // 选择公司
      const select = wrapper.find('select')
      await select.setValue('comp-1')
      await flushPromises()
      // 填 Git 用户名与邮箱
      const inputs = wrapper.findAll('input')
      const nameInput = inputs.find((i) => i.element.placeholder.includes('例如：alice'))
      await nameInput.setValue('alice')
      const emailInput = inputs.find((i) => i.element.placeholder.includes('alice@company.com'))
      await emailInput.setValue('alice@company.com')
      await flushPromises()
      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('新建身份'))
      await createBtn.trigger('click')
      await flushPromises()

      const postCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/git-identities/user/user-1/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
      expect(postCall[1].body).toContain('alice')
    })

    it('设默认身份 PATCH 携带 Idempotency-Key', async () => {
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'PATCH') return jsonOk({ ok: true })
        if (String(url).includes('/profile/')) {
          return jsonOk({ company_nicknames: [{ company_id: 'comp-1', company_name: '测试公司' }] })
        }
        return jsonOk({ identities: [{ id: 'id-1', company_id: 'comp-1', git_user_name: 'alice', is_default: false }] })
      })
      const wrapper = mount(View, {
        global: {
          stubs: {
            UserCenterSidebar: { template: '<div />' },
            'router-link': { template: '<a><slot /></a>' },
            'router-view': true,
          },
        },
      })
      await flushPromises()
      const defaultBtn = wrapper.findAll('button').find((b) => b.text().includes('设为默认'))
      await defaultBtn.trigger('click')
      await flushPromises()

      const patchCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/git-identities/user/user-1/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-test-1')
    })
  })
}
