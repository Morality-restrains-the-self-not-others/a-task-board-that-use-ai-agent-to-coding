if (!process.env.VITEST) {
  console.log('[skip] taskDetailLayerGraphModelOptions.test.js requires vitest runtime')
} else {
const { describe, it, expect, vi, beforeEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

vi.mock('../../utils/traceId.js', () => ({
  extractTraceId: () => '',
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const {
  createLayerGraphModelOptionsState,
  loadFeatureParamsPayloadForSource,
  resolveEffectiveFeatureParamsData,
} = await import('./taskDetailLayerGraphModelOptions.js')

function mockJsonResponse(body, ok = true) {
  return {
    ok,
    json: async () => body,
  }
}

describe('resolveEffectiveFeatureParamsData', () => {
  it('use_company_default 时取 company_config', () => {
    const data = resolveEffectiveFeatureParamsData({
      use_company_default: true,
      providers: [{ provider: 'ws', supported_models: ['w1'] }],
      agent_model: 'w1',
      agent_model_provider: 'ws',
      company_config: {
        providers: [{ provider: 'co', supported_models: ['c1', 'c2'] }],
        agent_model: 'c1',
        agent_model_provider: 'co',
      },
    })
    expect(data.agent_model).toBe('c1')
    expect(data.agent_model_provider).toBe('co')
    expect(data.providers[0].provider).toBe('co')
  })

  it('自定义 workspace 时取顶层字段', () => {
    const data = resolveEffectiveFeatureParamsData({
      use_company_default: false,
      providers: [{ provider: 'ws', supported_models: ['w1'] }],
      agent_model: 'w1',
      agent_model_provider: 'ws',
    })
    expect(data.agent_model).toBe('w1')
    expect(data.agent_model_provider).toBe('ws')
  })
})

describe('createLayerGraphModelOptionsState', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('company source 请求公司 feature-params', async () => {
    apiFetch.mockResolvedValue(mockJsonResponse({
      data: {
        providers: [{ provider: 'openai', supported_models: ['gpt-4'] }],
        agent_model: 'gpt-4',
        agent_model_provider: 'openai',
      },
    }))
    const state = createLayerGraphModelOptionsState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      localTask: ref({ feature_params_source: 'company' }),
      layerGraphCmdSending: ref(false),
      layerGraphCommandKind: ref('trae'),
    })
    await state.fetchLayerGraphModelOptions()
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/cloud/feature-params/tenant_id/t1?view=summary',
      expect.any(Object),
    )
    expect(state.layerGraphModelOptions.value).toEqual(['gpt-4'])
  })

  it('workspace source 请求工作空间 feature-params', async () => {
    apiFetch.mockResolvedValue(mockJsonResponse({
      data: {
        use_company_default: false,
        providers: [{ provider: 'deepseek', supported_models: ['deepseek-v3'] }],
        agent_model: 'deepseek-v3',
        agent_model_provider: 'deepseek',
      },
    }))
    const state = createLayerGraphModelOptionsState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      localTask: ref({ feature_params_source: 'workspace' }),
      layerGraphCmdSending: ref(false),
      layerGraphCommandKind: ref('trae'),
    })
    await state.fetchLayerGraphModelOptions()
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/cloud/feature-params/tenant_id/t1/workspace_id/w1?view=summary',
      expect.any(Object),
    )
    expect(state.layerGraphSelectedModel.value).toBe('deepseek-v3')
  })

  it('任务行持久化 agent_model 优先作为默认展示模型', async () => {
    apiFetch.mockResolvedValue(mockJsonResponse({
      data: {
        use_company_default: false,
        providers: [{ provider: 'openai', supported_models: ['gpt-4', 'gpt-4.1-mini'] }],
        agent_model: 'gpt-4',
        agent_model_provider: 'openai',
      },
    }))
    const state = createLayerGraphModelOptionsState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      localTask: ref({ feature_params_source: 'workspace', agent_model: 'gpt-4.1-mini' }),
      layerGraphCmdSending: ref(false),
      layerGraphCommandKind: ref('trae'),
    })
    await state.fetchLayerGraphModelOptions()
    expect(state.layerGraphModelOptions.value).toEqual(['gpt-4', 'gpt-4.1-mini'])
    expect(state.layerGraphSelectedModel.value).toBe('gpt-4.1-mini')
  })

  it('personal source 匹配个人配置 id', async () => {
    apiFetch.mockResolvedValue(mockJsonResponse({
      configs: [
        {
          id: 'cfg-a',
          providers: [{ provider: 'a', supported_models: ['m-a'] }],
          agent_model: 'm-a',
          agent_model_provider: 'a',
        },
        {
          id: 'cfg-b',
          providers: [{ provider: 'b', supported_models: ['m-b'] }],
          agent_model: 'm-b',
          agent_model_provider: 'b',
        },
      ],
    }))
    const state = createLayerGraphModelOptionsState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      localTask: ref({
        feature_params_source: 'personal',
        personal_feature_params_config_id: 'cfg-b',
      }),
      layerGraphCmdSending: ref(false),
      layerGraphCommandKind: ref('trae'),
    })
    await state.fetchLayerGraphModelOptions()
    expect(apiFetch).toHaveBeenCalledWith(
      '/api/personal/feature-params-configs/',
      expect.any(Object),
    )
    expect(state.layerGraphSelectedModel.value).toBe('m-b')
    expect(state.layerGraphModelOptions.value).toEqual(['m-b'])
  })

  it('company 502 HTML maps to retryable message instead of 获取模型配置失败', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: '796a1ffa-ede1-4a2f-988d-9566d706151d',
      _errorData: {
        _rawErrorText:
          '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
      },
      json: async () => ({}),
    })
    const state = createLayerGraphModelOptionsState({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('w1'),
      localTask: ref({ feature_params_source: 'company' }),
      layerGraphCmdSending: ref(false),
      layerGraphCommandKind: ref('trae'),
    })
    await state.fetchLayerGraphModelOptions()
    expect(state.layerGraphModelLoadError.value).toBe('服务暂时不可用，请稍后重试')
    expect(state.layerGraphModelLoadError.value).not.toBe('获取模型配置失败')
    expect(state.layerGraphModelLoadErrorTraceId.value).toBe('796a1ffa-ede1-4a2f-988d-9566d706151d')
  })
})

describe('loadFeatureParamsPayloadForSource', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('throws retryable message when company summary returns APISIX 502 HTML', async () => {
    apiFetch.mockResolvedValue({
      ok: false,
      status: 502,
      traceId: 'trace-502',
      _errorData: {
        _rawErrorText:
          '<html><head><title>502 Bad Gateway</title></head><body><h1>502 Bad Gateway</h1></body></html>',
      },
      json: async () => ({}),
    })
    await expect(
      loadFeatureParamsPayloadForSource({
        tenantId: 't1',
        workspaceId: 'w1',
        source: 'company',
      }),
    ).rejects.toMatchObject({
      message: '服务暂时不可用，请稍后重试',
      traceId: 'trace-502',
    })
    expect(apiFetch.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('recovers when company summary 502 is followed by 200', async () => {
    apiFetch
      .mockResolvedValueOnce({
        ok: false,
        status: 502,
        traceId: 'trace-502-first',
        _errorData: { _rawErrorText: '<html><h1>502 Bad Gateway</h1></html>' },
        json: async () => ({}),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: { get: () => null },
        json: async () => ({
          data: {
            providers: [{ provider: 'openai', supported_models: ['gpt-4'] }],
            agent_model: 'gpt-4',
            agent_model_provider: 'openai',
          },
        }),
      })
    const { data } = await loadFeatureParamsPayloadForSource({
      tenantId: 't1',
      workspaceId: 'w1',
      source: 'company',
    })
    expect(data.agent_model).toBe('gpt-4')
    expect(apiFetch).toHaveBeenCalledTimes(2)
  })
})

}
