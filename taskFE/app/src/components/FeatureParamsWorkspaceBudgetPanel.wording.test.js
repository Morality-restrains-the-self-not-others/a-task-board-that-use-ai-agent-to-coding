// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] FeatureParamsWorkspaceBudgetPanel.wording.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it, vi } = await import('vitest')

  vi.mock('../utils/apiUtils', () => ({
    apiFetch: vi.fn(async () => ({ ok: false })),
  }))

  describe('FeatureParamsWorkspaceBudgetPanel 文案', () => {
    it('说明文字不要求启用派生子 Key', async () => {
      const Comp = (await import('./FeatureParamsWorkspaceBudgetPanel.vue')).default
      const wrapper = mount(Comp, {
        props: { tenantId: 't1', enabled: true },
        global: {
          stubs: { WorkspaceSettingsLlmBudgetModal: { template: '<div />' } },
        },
      })
      expect(wrapper.text()).not.toContain('派生子 Key')
      expect(wrapper.text()).not.toContain('派生子Key')
    })
  })
}
