// @vitest-environment jsdom
// OPT-20260819-038 回归：访问管理 角色分配 PUT / 清理孤儿角色 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] PeopleAccess.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    clearCachedAuthToken: () => {},
  }))
  vi.mock('vue-router', () => ({
    useRoute: () => ({ params: { tenant: 't1' }, query: {} }),
    useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  }))
  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      hasRegionView: () => true,
      hasRegion: () => true,
      hasPage: () => true,
      hasPerm: () => true,
    }),
  }))
  vi.mock('./peopleAccessCatalog.js', () => ({
    fetchRoleBoundResourceGroups: vi.fn(async () => []),
    hasPeopleAccessWrite: () => true,
  }))
  vi.mock('../domain/auth/saveSubjectResourceAccess.js', () => ({
    listOrphanCustomAccessRoles: vi.fn(() => [{ id: 'r_orphan', name: 'orphan_role' }]),
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 PUT/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-access' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    mocks.apiFetch.mockImplementation((url) => {
      if (String(url).includes('company_members')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'm1', member_name: 'Alice' }]) })
      }
      if (String(url).includes('/groups')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
      }
      if (String(url).includes('member-role') || String(url).includes('group-role')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
      }
      if (String(url).includes('/roles/company_id/')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([{ id: 'r1', name: 'member', is_system: true, permissions: ['member:view'] }]) })
      }
      if (String(url).includes('resource-groups')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({}) })
    })
  })

  const { default: PeopleAccess } = await import('./PeopleAccess.vue')

  describe('PeopleAccess 写操作 clickGuard 接线', () => {
    it('保存角色分配 PUT 携带 Idempotency-Key', async () => {
      const wrapper = mount(PeopleAccess)
      await flushPromises()

      const memberBtn = wrapper.findAll('button').find((b) => b.text().includes('Alice'))
      expect(memberBtn).toBeTruthy()
      await memberBtn.trigger('click')
      await flushPromises()

      const saveBtn = wrapper.find('[data-testid="people-access-save"]')
      expect(saveBtn.exists()).toBe(true)
      await saveBtn.trigger('click')
      await flushPromises()

      const putCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'PUT')
      expect(putCall).toBeTruthy()
      expect(putCall[0]).toBe('/api/tenant/member-role/company_id/t1/member_id/m1/')
      expect(putCall[1].headers['Idempotency-Key']).toBe('ik-test-access')
    })

    it('清理孤儿角色 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(PeopleAccess)
      await flushPromises()

      const cleanupBtn = wrapper.find('[data-testid="cleanup-orphan-access-roles"]')
      expect(cleanupBtn.exists()).toBe(true)
      await cleanupBtn.trigger('click')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/auth/roles/role_id/r_orphan/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-access')
    })
  })
}
