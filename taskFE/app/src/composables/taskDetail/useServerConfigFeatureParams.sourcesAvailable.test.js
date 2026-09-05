// @vitest-environment jsdom
// OPT-20260809-018: 任务详情镜像区（ServerConfig.logic.vue 经 useServerConfigFeatureParams）
// 复用 create-task-modal 同源的可用性拉取：view=summary 消费后端 env_var_sources_available
// 标志 + 个人配置列表，计算三类来源是否至少一个可用；缺 workspace id 时降级公司级 GET。
// 未拉取/失败/缺标志时 fail-open 保持可用，避免误禁导致无法选择来源。
if (!process.env.VITEST) {
  console.log('[skip] useServerConfigFeatureParams.sourcesAvailable.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')
const { defineComponent, h } = await import('vue')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: hoisted.apiFetch,
}))

const { useServerConfigFeatureParams } = await import('./useServerConfigFeatureParams.js')

const okResp = (data) => ({
  ok: true,
  status: 200,
  json: async () => data,
})

function mountHarness(opts) {
  let result
  const Harness = defineComponent({
    setup() {
      result = useServerConfigFeatureParams(opts)
      return () => h('div')
    },
  })
  const wrapper = mount(Harness)
  return { wrapper, result }
}

const baseOpts = {
  props: { task: {} },
  emit: vi.fn(),
  route: { params: { tenant: 'c1' } },
  workspaceId: null,
}

beforeEach(() => {
  hoisted.apiFetch.mockReset()
})

describe('useServerConfigFeatureParams 来源可用性（OPT-20260809-018）', () => {
  it('有 workspace id：用 view=summary 拉取工作空间 feature-params，公司级 flag true → 可用', async () => {
    const { result } = mountHarness({
      ...baseOpts,
      workspaceId: 'ws1',
    })
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ data: { env_var_sources_available: { company: true, workspace: false } } }))
    })

    result.initFeatureParamsSource()
    await flushPromises()
    const [featureParamsUrl] = hoisted.apiFetch.mock.calls.find(([u]) => u.includes('feature-params/tenant_id/'))
    expect(featureParamsUrl).toContain('/workspace_id/ws1')
    expect(featureParamsUrl).toContain('?view=summary')
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('公司/工作空间标志全 false 且无个人配置 → 不可用（选择器应禁用）', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ data: { env_var_sources_available: { company: false, workspace: false } } }))
    })

    result.initFeatureParamsSource()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(false)
  })

  it('缺 workspace id → 降级公司级 GET（不含 workspace_id 段），公司级 flag true → 可用', async () => {
    const { result } = mountHarness(baseOpts) // workspaceId: null
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [] }))
      }
      return Promise.resolve(okResp({ data: { env_var_sources_available: { company: true, workspace: false } } }))
    })

    result.initFeatureParamsSource()
    await flushPromises()
    const [featureParamsUrl] = hoisted.apiFetch.mock.calls.find(([u]) => u.includes('feature-params/tenant_id/'))
    expect(featureParamsUrl).toContain('/tenant_id/c1?view=summary')
    expect(featureParamsUrl).not.toContain('/workspace_id/')
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('有个人配置即使公司/工作空间标志全 false → 可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockImplementation((url) => {
      if (url.includes('/api/personal/feature-params-configs/')) {
        return Promise.resolve(okResp({ configs: [{ id: 'pc1', name: '个人配置:mine' }] }))
      }
      return Promise.resolve(okResp({ data: { env_var_sources_available: { company: false, workspace: false } } }))
    })

    result.initFeatureParamsSource()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('缺 tenant id → 跳过不请求，保持可用（fail-open）', async () => {
    const { result } = mountHarness({
      ...baseOpts,
      route: { params: {} },
    })

    result.initFeatureParamsSource()
    await flushPromises()
    expect(hoisted.apiFetch).not.toHaveBeenCalled()
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

    result.initFeatureParamsSource()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })

  it('请求失败 → fail-open 保持可用', async () => {
    const { result } = mountHarness(baseOpts)
    hoisted.apiFetch.mockRejectedValue(new Error('network down'))

    result.initFeatureParamsSource()
    await flushPromises()
    expect(result.featureParamsSourcesAvailable.value).toBe(true)
  })
})
}
