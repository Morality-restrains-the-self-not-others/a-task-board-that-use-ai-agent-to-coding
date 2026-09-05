// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] FeatureParamsProvidersEditor.subTokenUi.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  describe('FeatureParamsProvidersEditor 派生子 Key 开关', () => {
    it('当前不展示「启用派生子Key」勾选', async () => {
      const Comp = (await import('./FeatureParamsProvidersEditor.vue')).default
      const providers = [{
        provider: 'openai',
        api_key: '',
        base_url: '',
        supported_models_text: 'gpt-4.1',
        use_sub_token: true,
        budget_enabled: true,
      }]
      const wrapper = mount(Comp, {
        props: {
          providers,
          subTokenProviders: [{ provider_name: 'openai' }],
        },
      })
      expect(wrapper.text()).not.toContain('启用派生子Key')
      expect(wrapper.text()).not.toContain('该供应商支持派生Token，已默认勾选')
      expect(providers[0].use_sub_token).toBe(false)
    })
  })
}
