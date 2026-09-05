// @vitest-environment jsdom
// OPT-20260809-019: fetchFeatureParamsSourcesAvailability 用 view=summary 拉取
// 并消费后端 env_var_sources_available 标志，避免拉取含 provider API key 的 full payload。
if (!process.env.VITEST) {
  console.log('[skip] useCreateTaskFeatureParams.sourcesAvailable.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')
const { defineComponent, h } = await import('vue')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
  warnNetworkFailure: vi.fn(),
  warnOptionalApiFailure: vi.fn(),
}))

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: hoisted.apiFetch,
}))

vi.mock('../utils/workPanelApiUtils.js', () => ({
  warnNetworkFailure: hoisted.warnNetworkFailure,
  warnOptionalApiFailure: hoisted.warnOptionalApiFailure,
}))

const { useCreateTaskFeatureParams } = await import('./useCreateTaskFeatureParams.js')

const okResp = (data) => ({
  ok: true,
  status: 200,
  json: async () => data,
})

function mountHarness(opts) {
  let result
  const Harness = defineComponent({
    setup() {
      result = useCreateTaskFeatureParams(opts)
      return () => h('div')
    },
  })
  const wrapper = mount(Harness)
  return { wrapper, result }
}

const baseOpts = {
  editingTask: () => ({}),
  currentWorkspace: () => ({ id: 'ws1' }),
  tenantId: () => 'c1',
  show: () => true,
}

beforeEach(() => {
  hoisted.apiFetch.mockReset()
  hoisted.warnNetworkFailure.mockClear()
  hoisted.warnOptionalApiFailure.mockClear()
})

describe('useCreateTaskFeatureParams 来源可用性（OPT-20260809-019）', () => {
  it('用 view=summary 拉取工作空间 feature-params', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ env_var_sources_available: { company: true, workspace: false } }))
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    const [featureParamsUrl] = hoisted.apiFetch.mock.calls.find(([u]) => u.includes('feature-params/tenant_id/'))
    expect(featureParamsUrl).toContain('?view=summary')
    expect(featureParamsUrl).not.toContain('X-Feature-Params-Access-Context')
    // 不再从 data.extra_env_vars 计算 key 结构（summary 已脱敏），仅消费标志
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('真实响应 data.env_var_sources_available 全 false 且无个人配置 → 不可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({
        data: { env_var_sources_available: { company: false, workspace: false } },
      }))
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(false)
  })

  it('公司/工作空间标志全 false 且无个人配置 → 不可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ env_var_sources_available: { company: false, workspace: false } }))
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(false)
  })

  it('公司级 flag true 但无个人配置 → 可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ env_var_sources_available: { company: true, workspace: false } }))
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('有个人配置即使公司/工作空间标志全 false → 可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [{ id: 'pc1', name: '个人配置:mine' }] }))
      }
      return Promise.resolve(okResp({ env_var_sources_available: { company: false, workspace: false } }))
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('响应缺标志（异常/兼容场景）→ fail-open 保持可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ data: { extra_env_vars: [] } }))
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('缺 workspace id → 跳过不请求', async () => {
    const { result } = mountHarness({
      ...baseOpts,
      currentWorkspace: () => null,
    })

    result.syncFeatureParamsFromEditingTask()
    await flushPromises()
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
  })
})
}
