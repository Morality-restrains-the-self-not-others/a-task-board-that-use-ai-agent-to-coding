// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] PeopleGroups.delete-confirm.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    confirmMock: vi.fn(),
    permsState: { t1: ['group-members:manage'] },
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

  // usePermissions 为模块级单例，直接 mock 掉以便逐测试注入权限码，避免跨用例污染
  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      hasPerm: (cid, code) =>
        Array.isArray(hoistedMocks.permsState[cid]) && hoistedMocks.permsState[cid].includes(code),
      hasRegion: () => false,
      hasPage: () => false,
      hasPlatformPerm: () => false,
      isPlatformRole: () => false,
    }),
  }))

  // 删除分组统一走自定义模态（替代浏览器 confirm），测试确认/取消两分支
  vi.mock('../utils/modalService.js', () => ({
    default: { confirm: (...args) => hoistedMocks.confirmMock(...args) },
  }))

  vi.mock('../utils/requestErrorDisplay.js', async (importOriginal) => {
    const actual = await importOriginal()
    return { ...actual, showRequestError: vi.fn() }
  })

  function okResponse(body) {
    return { ok: true, status: 200, headers: {}, json: async () => body }
  }

  function defaultApiImpl() {
    hoistedMocks.apiFetchMock.mockImplementation(async (url, opts = {}) => {
      if (url === '/api/accounts/users/me/') {
        return okResponse({
          companies: [{ id: 't1', name: 'ACME', is_admin: true }],
          current_company: { id: 't1' },
        })
      }
      if (url === '/api/tenant/t1/accounts/groups/') {
        return okResponse([{ id: 'g1', name: '研发组', memberCount: 2, description: 'desc' }])
      }
      if (url === '/api/tenant/t1/accounts/groups/g1/') {
        return okResponse({})
      }
      if (url === '/api/tenant/t1/accounts/groups/g1/members/') {
        return okResponse({ members: [] })
      }
      if (url === '/api/tenant/t1/accounts/members/company_members/') {
        return okResponse({ members: [] })
      }
      return {
        ok: false,
        status: 404,
        headers: {},
        json: async () => ({ message: 'unexpected url: ' + url }),
      }
    })
  }

  async function flushRender() {
    for (let i = 0; i < 12; i += 1) {
      await Promise.resolve()
      await nextTick()
    }
  }

  async function mountReady() {
    const Comp = (await import('./PeopleGroups.vue')).default
    const wrapper = mount(Comp)
    await flushRender()
    return wrapper
  }

  const deleteCalls = () =>
    hoistedMocks.apiFetchMock.mock.calls.filter(
      (c) => c[0].includes('/accounts/groups/') && c[1]?.method === 'DELETE',
    )

  describe('PeopleGroups 删除分组自定义确认（OPT-20260811-076）', () => {
    beforeEach(() => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.confirmMock.mockReset()
      hoistedMocks.permsState = { t1: ['group-members:manage'] }
      document.cookie = 'userId=test-user'
      defaultApiImpl()
    })

    it('确认删除：modalService.confirm resolve → 发 DELETE 到分组接口', async () => {
      hoistedMocks.confirmMock.mockResolvedValue(true)
      const wrapper = await mountReady()

      const groupCard = wrapper.find('.space-y-2 .cursor-pointer')
      expect(groupCard.exists()).toBe(true)
      await groupCard.trigger('click')
      await flushRender()

      const deleteBtn = wrapper.findAll('button').find((b) => b.text().includes('删除'))
      expect(deleteBtn).toBeTruthy()
      await deleteBtn.trigger('click')
      await flushRender()

      expect(hoistedMocks.confirmMock).toHaveBeenCalledWith('确定要删除该分组吗？')
      const del = deleteCalls()
      expect(del.length).toBe(1)
      expect(del[0][0]).toBe('/api/tenant/t1/accounts/groups/g1/')
    })

    it('取消删除：modalService.confirm reject → 不发 DELETE', async () => {
      hoistedMocks.confirmMock.mockRejectedValue(false)
      const wrapper = await mountReady()

      const groupCard = wrapper.find('.space-y-2 .cursor-pointer')
      await groupCard.trigger('click')
      await flushRender()

      const deleteBtn = wrapper.findAll('button').find((b) => b.text().includes('删除'))
      await deleteBtn.trigger('click')
      await flushRender()

      expect(hoistedMocks.confirmMock).toHaveBeenCalledWith('确定要删除该分组吗？')
      expect(deleteCalls().length).toBe(0)
    })
  })
}
