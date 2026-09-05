// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] InviteAccessGrants.test.js requires vitest runtime')
} else {
  const { mount, flushPromises } = await import('@vue/test-utils')
  const { beforeEach, describe, expect, it, vi } = await import('vitest')
  const { togglePageGrants } = await import('../domain/auth/resourceGrantEffects.js')

  const hoisted = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: (...args) => hoisted.apiFetchMock(...args),
  }))

  describe('InviteAccessGrants', () => {
    beforeEach(() => {
      hoisted.apiFetchMock.mockReset()
      hoisted.apiFetchMock.mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => null },
        json: async () => ({
          pages: [
            {
              id: 'rg-page-people-access',
              group_key: 'people.access',
              display_name: '访问管理',
              kind: 'page',
              children: [
                {
                  id: 'rg-reg-subject',
                  group_key: 'people.access.subject_list',
                  display_name: '主体列表',
                  kind: 'ui_region',
                },
                {
                  id: 'rg-reg-save',
                  group_key: 'people.access.save_actions',
                  display_name: '保存操作',
                  kind: 'ui_region',
                },
              ],
            },
          ],
        }),
      })
    })

    it('快捷「授予访问管理」发出 people.access 相关 grants', async () => {
      const Comp = (await import('./InviteAccessGrants.vue')).default
      const wrapper = mount(Comp, {
        props: { companyId: 'c1', disabled: false },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="invite-access-grants"]').exists()).toBe(true)

      await wrapper.find('[data-testid="invite-grant-access-mgmt"]').trigger('click')
      await flushPromises()

      const emitted = wrapper.emitted('update:grants')
      expect(emitted?.length).toBeGreaterThan(0)
      const last = emitted[emitted.length - 1][0]
      const keys = last.map((g) => g.group_key)
      expect(keys).toEqual(
        expect.arrayContaining([
          'people.access',
          'people.access.subject_list',
          'people.access.save_actions',
        ]),
      )
    })

    it('admin disabled 时不提交 grants', async () => {
      const Comp = (await import('./InviteAccessGrants.vue')).default
      const wrapper = mount(Comp, {
        props: { companyId: 'c1', disabled: true },
      })
      await flushPromises()
      expect(wrapper.find('[data-testid="invite-grants-admin-hint"]').exists()).toBe(true)
      const emitted = wrapper.emitted('update:grants')
      const last = emitted[emitted.length - 1][0]
      expect(last).toEqual([])
    })
  })

  describe('togglePageGrants people.access', () => {
    it('整页勾选写入全部 region operate', () => {
      const page = {
        group_key: 'people.access',
        children: [
          { group_key: 'people.access.subject_list' },
          { group_key: 'people.access.save_actions' },
        ],
      }
      const next = togglePageGrants({}, page, true)
      expect(next['people.access.subject_list']).toBe('operate')
      expect(next['people.access.save_actions']).toBe('operate')
    })
  })
}
