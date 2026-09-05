// @vitest-environment jsdom
// OPT-20260819-038 回归：分组 创建 POST / 删除 DELETE / 移除成员 DELETE 写请求携带 Idempotency-Key 头。
if (!process.env.VITEST) {
  console.log('[skip] PeopleGroups.click-guard.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    confirmMock: vi.fn(async () => true),
    routeMock: { params: { tenant: 't1' } },
    routerMock: { push: vi.fn() },
  }))

  vi.mock('../utils/apiUtils', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, apiFetch: hoistedMocks.apiFetchMock }
  })
  vi.mock('vue-router', () => ({
    useRoute: () => hoistedMocks.routeMock,
    useRouter: () => hoistedMocks.routerMock,
  }))
  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      hasPerm: () => true,
      hasRegion: () => true,
      hasPage: () => true,
      hasPlatformPerm: () => false,
      isPlatformRole: () => false,
    }),
  }))
  vi.mock('../utils/modalService.js', () => ({
    default: { confirm: (...args) => hoistedMocks.confirmMock(...args) },
  }))
  vi.mock('../utils/requestErrorDisplay.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, showRequestError: vi.fn() }
  })
  // OPT-20260819-038: 让 guard 直接放行并把幂等键透传给 mergeIdempotencyHeaders，
  // 断言 POST/DELETE 携带 Idempotency-Key 头。
  vi.mock('../utils/clickGuard.js', () => ({
    createClickGuard: () => ({
      run: async (fn) => fn({ idempotencyKey: 'ik-test-groups' }),
      isBusy: () => false,
    }),
    mergeIdempotencyHeaders: (headers, key) => ({ ...(headers || {}), 'Idempotency-Key': key }),
  }))

  function okResponse(body) {
    return { ok: true, status: 200, headers: {}, json: async () => body }
  }

  beforeEach(() => {
    vi.clearAllMocks()
    hoistedMocks.apiFetchMock.mockImplementation(async (url, opts = {}) => {
      if (url === '/api/accounts/users/me/') {
        return okResponse({ companies: [{ id: 't1', name: 'ACME', is_admin: true }] })
      }
      if (url.includes('/accounts/groups/') && opts?.method === 'POST') {
        return okResponse({ id: 'g1', name: '新分组' })
      }
      if (url.includes('/accounts/groups/') && opts?.method === 'DELETE') {
        return okResponse({})
      }
      if (url.includes('/accounts/groups/')) {
        return okResponse([{ id: 'g1', name: '开发组', description: '', member_count: 0 }])
      }
      if (url.includes('/accounts/company_members/')) {
        return okResponse({ members: [] })
      }
      return okResponse({})
    })
  })

  const { default: PeopleGroups } = await import('./PeopleGroups.vue')

  describe('PeopleGroups 写操作 clickGuard 接线', () => {
    it('创建分组 POST 携带 Idempotency-Key', async () => {
      const wrapper = mount(PeopleGroups)
      await flushPromises()

      const createBtn = wrapper.findAll('button').find((b) => b.text().includes('创建分组'))
      expect(createBtn).toBeTruthy()
      await createBtn.trigger('click')
      await flushPromises()

      await wrapper.find('#groupName').setValue('新分组')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      const postCall = hoistedMocks.apiFetchMock.mock.calls.find(([, o]) => o?.method === 'POST')
      expect(postCall).toBeTruthy()
      expect(postCall[0]).toBe('/api/tenant/t1/accounts/groups/')
      expect(postCall[1].headers['Idempotency-Key']).toBe('ik-test-groups')
    })

    it('删除分组 DELETE 携带 Idempotency-Key', async () => {
      const wrapper = mount(PeopleGroups)
      await flushPromises()

      const groupRow = wrapper.findAll('.cursor-pointer').find((el) => el.text().includes('开发组'))
      expect(groupRow).toBeTruthy()
      await groupRow.trigger('click')
      await flushPromises()

      const deleteBtn = wrapper.findAll('button').find((b) => b.text().trim() === '删除')
      expect(deleteBtn).toBeTruthy()
      await deleteBtn.trigger('click')
      await flushPromises()

      const delCall = hoistedMocks.apiFetchMock.mock.calls.find(([, o]) => o?.method === 'DELETE')
      expect(delCall).toBeTruthy()
      expect(delCall[0]).toBe('/api/tenant/t1/accounts/groups/g1/')
      expect(delCall[1].headers['Idempotency-Key']).toBe('ik-test-groups')
    })
  })
}
