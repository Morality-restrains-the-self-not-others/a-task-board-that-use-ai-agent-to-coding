// @vitest-environment jsdom
// OPT-20260819-038 回归：访问管理弹窗 移除成员/小组 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] AccessManagementModal.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { describe, expect, it, vi, beforeEach } = await import('vitest')

  const mocks = vi.hoisted(() => ({
    apiFetch: vi.fn(),
    showRequestError: vi.fn(),
  }))

  vi.mock('../utils/apiUtils.js', () => ({
    apiFetch: (...args) => mocks.apiFetch(...args),
    parseCompanyMembersResponse: (data) => ({ members: Array.isArray(data) ? data : [], meta: {} }),
  }))
  vi.mock('../utils/requestErrorDisplay.js', () => ({
    showRequestError: (...args) => mocks.showRequestError(...args),
  }))
  vi.mock('../utils/cookieUtils.js', () => ({ getCookie: () => 'x' }))
  vi.mock('./AccessMemberList.vue', () => ({
    default: { name: 'AccessMemberList', template: '<div />' },
  }))
  vi.mock('./AccessGroupList.vue', () => ({
    default: { name: 'AccessGroupList', template: '<div />' },
  }))
  vi.mock('./AccessAddMemberModal.vue', () => ({
    default: { name: 'AccessAddMemberModal', template: '<div />' },
  }))
  vi.mock('./AccessAddGroupModal.vue', () => ({
    default: { name: 'AccessAddGroupModal', template: '<div />' },
  }))
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-accessmodal' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  beforeEach(() => {
    vi.clearAllMocks()
    window.confirm = vi.fn(() => true)
    mocks.apiFetch.mockImplementation(async () => ({ ok: true, status: 200, json: async () => ({}) }))
  })

  const { default: AccessManagementModal } = await import('./AccessManagementModal.vue')
  const { default: AccessMemberList } = await import('./AccessMemberList.vue')
  const { default: AccessGroupList } = await import('./AccessGroupList.vue')

  describe('AccessManagementModal 写操作 clickGuard 接线', () => {
    it('移除成员 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(AccessManagementModal, {
        props: { show: true, workspaceId: 'ws1', tenantId: 't1' },
      })
      await flushPromises()

      const memberList = wrapper.findComponent(AccessMemberList)
      expect(memberList.exists()).toBe(true)
      await memberList.vm.$emit('remove', 'm1')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/projects/workspace-access/remove-permission/tenant_id/t1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-accessmodal')
    })

    it('移除小组 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(AccessManagementModal, {
        props: { show: true, workspaceId: 'ws1', tenantId: 't1' },
      })
      await flushPromises()

      const groupList = wrapper.findComponent(AccessGroupList)
      expect(groupList.exists()).toBe(true)
      await groupList.vm.$emit('remove', 'g1')
      await flushPromises()

      const delCall = mocks.apiFetch.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/projects/workspace-access/remove-permission/tenant_id/t1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-accessmodal')
    })
  })
}
