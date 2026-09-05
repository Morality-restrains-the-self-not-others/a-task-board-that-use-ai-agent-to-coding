// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] FeatureParamsProvidersEditor.deepSeekBaseUrl.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')

  const mkProviders = (overrides = {}) => [{
    provider: 'deepseek',
    api_key: '',
    base_url: '',
    supported_models_text: 'deepseek-chat',
    use_sub_token: false,
    budget_enabled: false,
    ...overrides,
  }]

  describe('FeatureParamsProvidersEditor 不按运营商改写 base_url', () => {
    it('改供应商名不预填 base_url', async () => {
      const Comp = (await import('./FeatureParamsProvidersEditor.vue')).default
      const providers = mkProviders()
      const wrapper = mount(Comp, { props: { providers, subTokenProviders: [] } })

      const nameInput = wrapper.find('input[placeholder="例如：openai"]')
      await nameInput.setValue('deepseek')
      await nameInput.trigger('change')

      expect(providers[0].base_url).toBe('')
    })

    it('已配置自定义 base_url 时不被覆盖', async () => {
      const Comp = (await import('./FeatureParamsProvidersEditor.vue')).default
      const providers = mkProviders({ base_url: 'https://gateway.example.com/v1' })
      const wrapper = mount(Comp, { props: { providers, subTokenProviders: [] } })

      const nameInput = wrapper.find('input[placeholder="例如：openai"]')
      await nameInput.trigger('change')

      expect(providers[0].base_url).toBe('https://gateway.example.com/v1')
    })

    it('任意 /anthropic 路径原样保留且不告警改写', async () => {
      const Comp = (await import('./FeatureParamsProvidersEditor.vue')).default
      const providers = mkProviders({ base_url: 'https://api.deepseek.com/anthropic' })
      const wrapper = mount(Comp, { props: { providers, subTokenProviders: [] } })

      expect(wrapper.text()).not.toContain('检测到 /anthropic 路径')
      expect(providers[0].base_url).toBe('https://api.deepseek.com/anthropic')
    })

    it('失焦时拆开两个 https:// 粘在一起的脏数据', async () => {
      const Comp = (await import('./FeatureParamsProvidersEditor.vue')).default
      const providers = mkProviders({
        base_url: 'https://api.deepseek.com/v1https://api.deepseek.com',
      })
      const wrapper = mount(Comp, { props: { providers, subTokenProviders: [] } })
      const urlInput = wrapper.find('input[placeholder="例如：https://api.openai.com/v1"]')
      await urlInput.trigger('blur')
      expect(providers[0].base_url).toBe('https://api.deepseek.com')
    })
  })
}
