// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] MemberList.click-guard.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const { apiFetch } = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch,
    parseCompanyMembersResponse: (data) => {
      if (Array.isArray(data)) {
        return { members: data, meta: {} }
      }
      return {
        members: Array.isArray(data?.members) ? data.members : [],
        meta: data?.meta && typeof data.meta === 'object' ? data.meta : {},
      }
    },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PATCH/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-member' }),
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))
  vi.mock('./MemberGitIdentitiesModal.vue', () => ({
    default: { template: '<div />' },
  }))

  const { default: Component } = await import('./MemberList.vue')

  function jsonOk(body) {
    return {
      ok: true,
      json: async () => body,
      headers: { get: () => 'application/json' },
    }
  }

  const MEMBERS_PAYLOAD = {
    members: [
      { id: 'm-1', name: 'alice', email: 'alice@example.com', status: 'active', role: 'member', avatar_url: '' },
    ],
    meta: {
      has_permission: true,
      llm_budget_enabled: false,
      can_manage_budget_permissions: false,
      budget_raise_by_member_id: {},
    },
  }

  describe('MemberList 写操作 clickGuard 接线', () => {
    beforeEach(() => {
      window.confirm = vi.fn(() => true)
      apiFetch.mockReset()
      apiFetch.mockImplementation(async (url, opts) => {
        if (opts?.method === 'GET' || !opts?.method) {
          return jsonOk(MEMBERS_PAYLOAD)
        }
        return jsonOk({})
      })
    })

    it('禁用成员 PATCH 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, { props: { tenantId: 'tenant-1' } })
      await flushPromises()

      const disableBtn = wrapper.findAll('button').find((b) => b.text().trim() === '禁用')
      expect(disableBtn).toBeTruthy()
      await disableBtn.trigger('click')
      await flushPromises()

      const patchCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'PATCH')
      expect(patchCall).toBeTruthy()
      expect(patchCall[0]).toBe('/api/tenant/tenant-1/accounts/members/m-1/toggle_status/')
      expect(patchCall[1].headers['Idempotency-Key']).toBe('ik-test-member')
    })

    it('移除成员 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(Component, { props: { tenantId: 'tenant-1' } })
      await flushPromises()

      const removeBtn = wrapper.findAll('button').find((b) => b.text().trim() === '移除')
      expect(removeBtn).toBeTruthy()
      await removeBtn.trigger('click')
      await flushPromises()

      const delCall = apiFetch.mock.calls.find(([, opts]) => opts?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/tenant/tenant-1/accounts/members/m-1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-member')
    })
  })
}
