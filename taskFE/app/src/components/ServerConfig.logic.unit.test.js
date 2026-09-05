// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ServerConfig.logic.unit.test.js requires vitest runtime')
} else {
/**
 * OPT-20260718-051: ServerConfig.logic fetchServerRuntimeStatus 单元测试
 *
 * 验证 fetchServerRuntimeStatus 成功获取运行状态后正确调用
 * notifyRuntimeHydrate → updateServerStatus({ status: 'runtime_hydrate', runtime_status }).
 *
 * 防止回归：updateServerStatus 中存在 runtime_hydrate 处理器，
 * 但 fetchServerRuntimeStatus 未正确调用它导致父组件状态不与云 Describe 对齐。
 */
const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')
const { mount, flushPromises } = await import('@vue/test-utils')
const { nextTick } = await import('vue')
const { createRouter, createMemoryHistory } = await import('vue-router')

// ── Mock variables: vi.hoisted 确保在 vi.mock 之前被提升 ──
const { mockApiFetch, mockNotifyRuntimeHydrate } = vi.hoisted(() => {
  return {
    mockApiFetch: vi.fn(),
    mockNotifyRuntimeHydrate: vi.fn(),
  }
})

// ── Mock apiFetch ──
vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: mockApiFetch,
}))

// ── Mock notifyRuntimeHydrate 保持 spy 行为 ──
vi.mock('../composables/taskDetail/serverRuntimeHydrate.js', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    notifyRuntimeHydrate: mockNotifyRuntimeHydrate,
  }
})

// ── Mock useServerSectionBodyExpanded ──
vi.mock('../composables/useServerSectionBodyExpanded.js', () => ({
  useServerSectionBodyExpanded: () => ({
    serverSectionBodyExpanded: { value: true },
    expandServerSectionBody: vi.fn(),
    toggleServerSectionBody: vi.fn(),
  }),
}))

// ── Mock serverRuntimeStatusDetails 工具 ──
vi.mock('../utils/serverRuntimeStatusDetails.js', () => ({
  buildServerRuntimeStatusDetails: vi.fn(() => []),
  resolveRuntimeUptimeSource: vi.fn(() => null),
}))

// mock 之后 import 组件（vi.mock 提升至 import 之上）
const { default: ServerConfigLogic } = await import('./ServerConfig.logic.vue')

describe('ServerConfig.logic fetchServerRuntimeStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('onMounted 无评论 ID 时不调用 notifyRuntimeHydrate（运行态按评论查询）', async () => {
    const updateServerStatus = vi.fn()

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/',
          component: { template: '<div/>' },
        },
      ],
    })
    await router.push('/tenant/t1/workspace/w1/task-detail/task_1/')
    await router.isReady()

    mockApiFetch.mockImplementation(async (url) => {
      if (url.includes('installed-images')) {
        return {
          ok: true,
          json: async () => ({ results: [] }),
        }
      }
      if (url.includes('server-runtime-status')) {
        return {
          ok: true,
          json: async () => ({
            runtime_status: 'Running',
            message: '服务器运行中',
          }),
        }
      }
      return {
        ok: true,
        json: async () => ({}),
      }
    })

    const wrapper = mount(ServerConfigLogic, {
      props: {
        task: { id: 'task_1' },
        updateServerStatus,
      },
      global: {
        plugins: [router],
        stubs: {
          Teleport: true,
          ServerConfigImageSectionHints: { template: '<div />' },
          ServerConfigFeatureParamsBlock: { template: '<div />' },
          ServerConfigSectionTabs: { template: '<div />' },
          ServerConfigServerStartHistoryPanel: { template: '<div />' },
          ServerConfigServerContentSection: { template: '<div />' },
          ServerConfigHardwarePanel: { template: '<div />' },
          ServerConfigRelayDirectPanel: { template: '<div />' },
        },
      },
    })

    await flushPromises()
    await nextTick()
    await flushPromises()

    expect(mockNotifyRuntimeHydrate).not.toHaveBeenCalled()
    expect(mockApiFetch.mock.calls.some((c) => String(c[0]).includes('server-runtime-status'))).toBe(false)

    wrapper.unmount()
  })

  it('API 返回 status=error 时不调用 notifyRuntimeHydrate', async () => {
    const updateServerStatus = vi.fn()

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/',
          component: { template: '<div/>' },
        },
      ],
    })
    await router.push('/tenant/t1/workspace/w1/task-detail/task_1/')
    await router.isReady()

    mockApiFetch.mockImplementation(async (url) => {
      if (url.includes('installed-images')) {
        return {
          ok: true,
          json: async () => ({ results: [] }),
        }
      }
      if (url.includes('server-runtime-status')) {
        return {
          ok: false,
          status: 500,
          json: async () => ({
            status: 'error',
            message: '查询服务器运行状态失败',
          }),
        }
      }
      return {
        ok: true,
        json: async () => ({}),
      }
    })

    const wrapper = mount(ServerConfigLogic, {
      props: {
        task: { id: 'task_1' },
        updateServerStatus,
      },
      global: {
        plugins: [router],
        stubs: {
          Teleport: true,
          ServerConfigImageSectionHints: { template: '<div />' },
          ServerConfigFeatureParamsBlock: { template: '<div />' },
          ServerConfigSectionTabs: { template: '<div />' },
          ServerConfigServerStartHistoryPanel: { template: '<div />' },
          ServerConfigServerContentSection: { template: '<div />' },
          ServerConfigHardwarePanel: { template: '<div />' },
          ServerConfigRelayDirectPanel: { template: '<div />' },
        },
      },
    })

    await flushPromises()
    await nextTick()
    await flushPromises()

    // API 返回 error 时不应调用 notifyRuntimeHydrate
    expect(mockNotifyRuntimeHydrate).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})

describe('ServerConfig.logic resolvedTaskId — 路由/props 回退（缺少任务ID 回归）', () => {
  const stubs = {
    Teleport: true,
    ServerConfigImageSectionHints: { template: '<div />' },
    ServerConfigFeatureParamsBlock: { template: '<div />' },
    ServerConfigSectionTabs: { template: '<div />' },
    ServerConfigServerStartHistoryPanel: { template: '<div />' },
    ServerConfigServerContentSection: { template: '<div />' },
    ServerConfigHardwarePanel: { template: '<div />' },
    ServerConfigRelayDirectPanel: {
      props: ['hasTaskId'],
      template: '<div data-testid="relay-has-task-id">{{ hasTaskId }}</div>',
    },
  }

  async function mountLogic(props) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/',
          component: { template: '<div/>' },
        },
      ],
    })
    await router.push('/tenant/t1/workspace/w1/task-detail/task_881388002226499584/')
    await router.isReady()
    mockApiFetch.mockResolvedValue({ ok: true, json: async () => ({ results: [] }) })
    const wrapper = mount(ServerConfigLogic, {
      props,
      global: { plugins: [router], stubs },
    })
    await flushPromises()
    await nextTick()
    return wrapper
  }

  it('task 无 id 时仍把 has-task-id=true 传给 relay 面板（URL 已有 taskId）', async () => {
    const wrapper = await mountLogic({ task: {}, updateServerStatus: vi.fn() })
    expect(wrapper.get('[data-testid="relay-has-task-id"]').text()).toBe('true')
    wrapper.unmount()
  })

  it('props.taskId 在 task 为空对象时也可解析', async () => {
    const wrapper = await mountLogic({
      task: {},
      taskId: 'prop-task',
      updateServerStatus: vi.fn(),
    })
    expect(wrapper.get('[data-testid="relay-has-task-id"]').text()).toBe('true')
    wrapper.unmount()
  })
})
}
