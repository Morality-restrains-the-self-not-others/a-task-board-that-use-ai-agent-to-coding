// @vitest-environment jsdom
/**
 * OPT-20260811-082: 邀请页可复用角色多选（v75 角色优先）
 *
 * 1) 成员邀请显示可复用角色勾选项并加载角色列表
 * 2) 勾选角色后发送邀请 body 含 role_names
 * 3) 管理员邀请不显示角色勾选项，且 body 不含 role_names
 */
if (!process.env.VITEST) {
  console.log('[skip] PeopleInvite.role-options.test.js requires vitest runtime')
} else {
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { mount, flushPromises } = await import('@vue/test-utils')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
    routeMock: { params: { tenant: 't1' } },
    routerMock: { push: vi.fn() },
    toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  vi.mock('../utils/toastService', () => ({
    default: hoisted.toast,
  }))

  vi.mock('../composables/usePermissions.js', () => ({
    usePermissions: () => ({
      load: vi.fn(async () => {}),
      reload: vi.fn(async () => {}),
      hasPerm: (cid, code) => code === 'member:manage',
      hasRegion: () => true,
      hasPage: () => true,
    }),
  }))

  vi.mock('vue-router', () => ({
    useRoute: () => hoisted.routeMock,
    useRouter: () => hoisted.routerMock,
  }))

  function okResponse(body) {
    return { ok: true, status: 200, json: async () => body }
  }

  function okCreated(body) {
    return { ok: true, status: 201, json: async () => body }
  }

  function defaultApiImpl() {
    hoisted.apiFetchMock.mockImplementation(async (url, opts) => {
      if (url.startsWith('/api/projects/workspaces/tenant_id/')) {
        return okResponse({ data: [{ id: 'ws1', name: 'WS1' }] })
      }
      if (url.startsWith('/api/auth/roles/company_id/')) {
        return okResponse([
          { id: 'r1', name: 'deploy_engineer', display_name: '部署工程师', is_system: false },
          { id: 'r2', name: 'readonly', display_name: '只读', is_system: false },
        ])
      }
      if (url.startsWith('/api/auth/resource-groups/')) {
        return okResponse({ pages: [] })
      }
      if (url.includes('/accounts/members/invite/')) {
        return okCreated({ message: '邀请链接已生成', invite_token: 'tok123' })
      }
      return okResponse({})
    })
  }

  const Comp = async () => (await import('./PeopleInvite.vue')).default

  describe('PeopleInvite 可复用角色多选', () => {
    beforeEach(() => {
      vi.clearAllMocks()
      hoisted.toast.success.mockReset()
      hoisted.toast.error.mockReset()
      hoisted.toast.warning.mockReset()
      hoisted.routeMock.params = { tenant: 't1' }
      // PeopleInvite 通过 main.js 的 window 全局调用 apiFetch（非 apiUtils import）
      window.apiFetch = hoisted.apiFetchMock
      defaultApiImpl()
    })

    it('成员邀请渲染角色勾选项并加载角色列表', async () => {
      const wrapper = mount(await Comp(), {
        global: { stubs: { InviteAccessGrants: true } },
      })
      await flushPromises()
      await flushPromises()

      const roleOptions = wrapper.find('[data-testid="people-invite-role-options"]')
      expect(roleOptions.exists()).toBe(true)
      expect(wrapper.find('[data-testid="people-invite-role-deploy_engineer"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="people-invite-role-readonly"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('部署工程师')
    })

    it('勾选角色后发送邀请 body 含 role_names', async () => {
      const wrapper = mount(await Comp(), {
        global: { stubs: { InviteAccessGrants: true } },
      })
      await flushPromises()
      await flushPromises()

      await wrapper.find('[data-testid="people-invite-role-deploy_engineer"]').setValue(true)
      await wrapper.find('[data-testid="people-invite-role-readonly"]').setValue(true)

      await wrapper.find('#email').setValue('invitee@example.com')
      await wrapper.find('#company_member_name').setValue('Invitee')
      await wrapper.find('form').trigger('submit.prevent')
      await flushPromises()

      const inviteCall = hoisted.apiFetchMock.mock.calls.find(([url]) => url.includes('/accounts/members/invite/'))
      expect(inviteCall).toBeTruthy()
      const body = JSON.parse(inviteCall[1].body)
      expect(body.role_names).toEqual(['deploy_engineer', 'readonly'])
      expect(hoisted.toast.success).toHaveBeenCalled()
    })

    it('管理员邀请隐藏角色勾选项且 body 不含 role_names', async () => {
      const wrapper = mount(await Comp(), {
        global: { stubs: { InviteAccessGrants: true } },
      })
      await flushPromises()
      await flushPromises()

      // 切换为管理员
      const roleSelect = wrapper.find('#role-shared')
      await roleSelect.setValue('admin')

      expect(wrapper.find('[data-testid="people-invite-role-options"]').exists()).toBe(false)

      await wrapper.find('#email').setValue('admin@example.com')
      await wrapper.find('#company_member_name').setValue('AdminInvitee')
      await wrapper.find('form').trigger('submit.prevent')
      await flushPromises()

      const inviteCall = hoisted.apiFetchMock.mock.calls.find(([url]) => url.includes('/accounts/members/invite/'))
      expect(inviteCall).toBeTruthy()
      const body = JSON.parse(inviteCall[1].body)
      expect(body.role).toBe('admin')
      expect(body.role_names).toBeUndefined()
    })
  })
}
