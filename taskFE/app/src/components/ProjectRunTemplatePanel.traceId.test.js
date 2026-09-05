// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ProjectRunTemplatePanel.traceId.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi } = await import('vitest')
const { mount } = await import('@vue/test-utils')

const hoisted = vi.hoisted(() => ({
  apiFetch: vi.fn(),
}))
const apiFetch = hoisted.apiFetch

vi.mock('../utils/apiUtils.js', () => ({
  apiFetch: (...args) => hoisted.apiFetch(...args),
}))
vi.mock('./ServerConfigHardwarePanel.vue', () => ({
  default: { template: '<div />' },
}))
vi.mock('../utils/workspaceCloudPlatformsApi.js', () => ({
  fetchWorkspaceCloudPlatforms: async () => [],
}))

const { default: ProjectRunTemplatePanel } = await import('./ProjectRunTemplatePanel.vue')

function jsonResponse(body, { ok = true, status = 200, headers = {} } = {}) {
  return {
    ok,
    status,
    headers: { get: (k) => headers[k] || headers[String(k).toLowerCase()] || null },
    json: async () => body,
  }
}

describe('ProjectRunTemplatePanel data-traceId', () => {
  it('错误元素携带 data-traceId（来自失败 PATCH 响应头）', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse(
        { detail: 'method not allowed' },
        { ok: false, status: 405, headers: { 'X-Trace-Id': 'tr-panel-save' } },
      ),
    )
    const wrapper = mount(ProjectRunTemplatePanel, {
      props: {
        tenantId: 't1',
        projectId: 'p1',
        project: { workspaces: ['ws_1'], server_run_template: {} },
      },
    })
    // 让硬件面板 payload 非空：面板内部 stubbed，buildRunTemplatePayload 缺失 →
    // handleSave 走空 payload 分支报「请至少选择…」。直接触发保存需先使 payload 可用。
    // 改用 composable 共享 error ref 的方式验证不可行（内部实现）；此处验证成功/失败路径的
    // data-traceId 绑定逻辑：直接向组件暴露的错误 ref 写入值（同一响应式源）。
    const panel = wrapper.vm
    panel.hardwarePanelRef = {
      buildRunTemplatePayload: () => ({ platform_type: 'aliyun', region: 'cn-hangzhou' }),
    }
    await wrapper.find('button').trigger('click')
    await vi.waitFor(() => {
      expect(wrapper.find('.text-red-600').exists()).toBe(true)
    })
    const el = wrapper.find('.text-red-600').element
    expect(el.getAttribute('data-traceId')).toBe('tr-panel-save')
    expect(el.textContent).toContain('method not allowed')
  })

  it('无 traceId 时错误元素不渲染 data-traceId 属性', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse({ detail: 'boom' }, { ok: false, status: 500 }),
    )
    const wrapper = mount(ProjectRunTemplatePanel, {
      props: {
        tenantId: 't1',
        projectId: 'p1',
        project: { workspaces: ['ws_1'], server_run_template: {} },
      },
    })
    const panel = wrapper.vm
    panel.hardwarePanelRef = {
      buildRunTemplatePayload: () => ({ platform_type: 'aliyun', region: 'cn-hangzhou' }),
    }
    await wrapper.find('button').trigger('click')
    await vi.waitFor(() => {
      expect(wrapper.find('.text-red-600').exists()).toBe(true)
    })
    const el = wrapper.find('.text-red-600').element
    expect(el.hasAttribute('data-traceId')).toBe(false)
  })

  // 真实网关格式（2026-08-07 真机复验）：X-Trace-Id 为 "id1, id2" 双值合并串，
  // data-traceId 必须取首段 id1。
  it('错误元素 data-traceId 取双值 X-Trace-Id 的首段', async () => {
    apiFetch.mockImplementation(async () =>
      jsonResponse(
        { detail: 'method not allowed' },
        {
          ok: false,
          status: 405,
          headers: { 'X-Trace-Id': 'ef4fc3d811c51cc9351f8275f927afc1, ef4fc3d811c51cc9351f8275f927afc1' },
        },
      ),
    )
    const wrapper = mount(ProjectRunTemplatePanel, {
      props: {
        tenantId: 't1',
        projectId: 'p1',
        project: { workspaces: ['ws_1'], server_run_template: {} },
      },
    })
    const panel = wrapper.vm
    panel.hardwarePanelRef = {
      buildRunTemplatePayload: () => ({ platform_type: 'aliyun', region: 'cn-hangzhou' }),
    }
    await wrapper.find('button').trigger('click')
    await vi.waitFor(() => {
      expect(wrapper.find('.text-red-600').exists()).toBe(true)
    })
    const el = wrapper.find('.text-red-600').element
    expect(el.getAttribute('data-traceId')).toBe('ef4fc3d811c51cc9351f8275f927afc1')
  })
})
}
