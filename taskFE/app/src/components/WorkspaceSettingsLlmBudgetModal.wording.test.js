// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] WorkspaceSettingsLlmBudgetModal.wording.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { nextTick } = await import('vue')
  const { describe, expect, it, vi } = await import('vitest')

  const hoistedMocks = vi.hoisted(() => ({
    apiFetchMock: vi.fn(),
  }))

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: hoistedMocks.apiFetchMock,
  }))

  describe('WorkspaceSettingsLlmBudgetModal 文案', () => {
    it('空列表不引导用户去环境变量页启用派生子 Key', async () => {
      hoistedMocks.apiFetchMock.mockReset()
      hoistedMocks.apiFetchMock.mockResolvedValue({
        ok: true,
        json: async () => ({ items: [] }),
      })
      const Comp = (await import('./WorkspaceSettingsLlmBudgetModal.vue')).default
      const wrapper = mount(Comp, {
        props: {
          visible: true,
          tenantId: '875588283562749952',
          workspaceId: 'ws-1',
          workspaceLabel: '默认空间',
        },
        global: {
          stubs: { 'router-link': { template: '<a><slot /></a>' } },
        },
      })
      for (let i = 0; i < 8; i += 1) {
        await Promise.resolve()
        await nextTick()
      }
      expect(wrapper.text()).not.toContain('派生子 Key')
      expect(wrapper.text()).not.toContain('派生子Key')
      expect(wrapper.text()).toContain('没有可配置预算的模型端点')
    })
  })
}
