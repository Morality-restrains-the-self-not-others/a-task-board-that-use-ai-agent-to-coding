// @vitest-environment jsdom
// OPT-20260819-038 回归：删除角色 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] PeopleRoles.click-guard.test.js requires vitest runtime')
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
      hasPage: () => true,
      hasRegion: () => true,
      hasRegionOperate: () => true,
      hasPerm: () => true,
    }),
  }))
  vi.mock('../domain/auth/tenantConsoleNav.js', () => ({
    isSystemTenantRole: () => false,
  }))
  vi.mock('./peopleAccessCatalog.js', () => ({
    fetchRoleBoundResourceGroups: vi.fn(async () => []),
    filterCatalogPages: (pages) => pages,
  }))
  vi.mock('../domain/auth/saveRoleResourceGrants.js', () => ({
    createTenantRole: vi.fn(async () => ({ id: 'r_new' })),
    saveRoleResourceGrants: vi.fn(async () => {}),
  }))
  vi.mock('../components/ResourceGrantMatrix.vue', () => ({
    default: { template: '<div />' },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-role' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    mocks.apiFetch.mockImplementation((url) => {
      if (String(url).includes('/roles/company_id/')) {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([{ id: 'r1', name: 'custom_role', display_name: '测试角色', is_system: false }]),
        })
      }
      if (String(url).includes('member-role') || String(url).includes('group-role')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
      }
      if (String(url).includes('resource-groups')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve({ pages: [] }) })
      }
      return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve({}) })
    })
  })

  const { default: PeopleRoles } = await import('./PeopleRoles.vue')

  describe('PeopleRoles 写操作 clickGuard 接线', () => {
    it('删除角色 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(PeopleRoles)
      await flushPromises()

      const roleBtn = wrapper.findAll('button').find((b) => b.text().includes('测试角色'))
      expect(roleBtn).toBeTruthy()
      await roleBtn.trigger('click')
      await flushPromises()

      const deleteBtn = wrapper.find('[data-testid="people-roles-delete"]')
      expect(deleteBtn.exists()).toBe(true)
      await deleteBtn.trigger('click')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/auth/roles/role_id/r1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-role')
    })
  })
}
