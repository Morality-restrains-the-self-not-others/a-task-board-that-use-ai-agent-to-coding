// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] agentModelOptions.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    buildAgentModelOptionsFromFeatureParams,
    preselectDefaultAgentModel,
  } = await import('./agentModelOptions.js')

  describe('buildAgentModelOptionsFromFeatureParams', () => {
    it('lists supported_models of agent_model_provider and prepends default', () => {
      const got = buildAgentModelOptionsFromFeatureParams({
        agent_model_provider: 'openai',
        agent_model: 'gpt-4.1',
        providers: [
          { provider: 'openai', supported_models: ['gpt-4.1-mini', 'gpt-4.1'] },
          { provider: 'other', supported_models: ['skip-me'] },
        ],
      })
      expect(got.provider).toBe('openai')
      expect(got.defaultModel).toBe('gpt-4.1')
      expect(got.options).toEqual(['gpt-4.1-mini', 'gpt-4.1'])
    })

    it('prepends default when it is missing from supported_models', () => {
      const got = buildAgentModelOptionsFromFeatureParams({
        agent_model_provider: 'openai',
        agent_model: 'gpt-4.1',
        providers: [{ provider: 'openai', supported_models: ['gpt-4.1-mini'] }],
      })
      expect(got.options[0]).toBe('gpt-4.1')
      expect(got.options).toContain('gpt-4.1-mini')
    })

    it('returns empty options when provider has no models', () => {
      const got = buildAgentModelOptionsFromFeatureParams({
        agent_model_provider: '',
        agent_model: '',
        providers: [],
      })
      expect(got.options).toEqual([])
    })
  })

  describe('preselectDefaultAgentModel', () => {
    it('pre-checks default when present', () => {
      expect(preselectDefaultAgentModel(['a', 'b'], 'b')).toEqual(['b'])
    })

    it('returns empty when default missing', () => {
      expect(preselectDefaultAgentModel(['a'], 'z')).toEqual([])
    })
  })
}
