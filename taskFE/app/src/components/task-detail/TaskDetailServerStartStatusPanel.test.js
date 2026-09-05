// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailServerStartStatusPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailServerStartStatusPanel } = await import('./TaskDetailServerStartStatusPanel.vue')

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { tenant: 't1' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

function mountPanel(overrides = {}) {
  return mount(TaskDetailServerStartStatusPanel, {
    props: {
      serverStatus: 'success',
      isServerRunning: true,
      isServerStarting: false,
      sseLive: true,
      sseReconnecting: false,
      sseReconnectAttempts: 0,
      statusProgress: 100,
      statusMessage: '服务器已就绪',
      statusLogs: [],
      ...overrides,
    },
  })
}

describe('TaskDetailServerStartStatusPanel SSE vs server lifecycle', () => {
  it('T1: SSE 文案为「SSE 已连接」且旁侧 label 为 SSE 连接，不再使用「实时推送」', () => {
    const wrapper = mountPanel()
    const sseRow = wrapper.get('[data-testid="sse-connection-row"]')
    expect(sseRow.text()).toContain('SSE 连接')
    expect(sseRow.get('[data-testid="sse-connection-status"]').text()).toBe('SSE 已连接')
    expect(wrapper.text()).not.toContain('实时推送')
  })

  it('T2: 服务器已启动时，「服务器启动状态」旁标注「已启动」，与 SSE 分行', () => {
    const wrapper = mountPanel({ isServerRunning: true, sseLive: true })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('已启动')
    expect(wrapper.get('[data-testid="sse-connection-status"]').text()).toBe('SSE 已连接')
    expect(wrapper.get('[data-testid="server-lifecycle-row"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sse-connection-row"]').exists()).toBe(true)
  })

  it('T3: SSE 已连接但服务器未启动时，生命周期为「未启动」/「启动中」可独立于 SSE', () => {
    const wrapper = mountPanel({
      serverStatus: 'processing',
      isServerRunning: false,
      isServerStarting: true,
      sseLive: true,
      statusProgress: 20,
      statusMessage: '正在创建实例',
    })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('启动中')
    expect(wrapper.get('[data-testid="sse-connection-status"]').text()).toBe('SSE 已连接')
    expect(
      wrapper.get('[data-testid="sse-connection-status"]').attributes('aria-label'),
    ).toBe('启动状态 SSE 已连接')
  })

  it('T3b: runtime Running + starting → 「等待容器」（VM 已起、容器未登记）', () => {
    const wrapper = mountPanel({
      serverStatus: 'processing',
      isServerRunning: false,
      isServerStarting: true,
      runtimeStatus: 'Running',
      sseLive: true,
      statusProgress: 60,
      statusMessage: '云主机已运行，等待容器登记可达地址',
    })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('等待容器')
    expect(wrapper.get('[data-testid="sse-connection-status"]').text()).toBe('SSE 已连接')
  })

  it('T3c: runtime Running + serverStatus error → 「启动失败」+ 可达超时横幅', () => {
    const wrapper = mountPanel({
      serverStatus: 'error',
      isServerRunning: false,
      isServerStarting: false,
      runtimeStatus: 'Running',
      sseLive: true,
      statusMessage: '云主机已运行，但容器服务未在时限内登记可达地址。不是阿里云 API 连不上：实例已 Running，请检查镜像 UserData / 安全组出站 / 容器 HTTP 进程后重试。',
    })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('启动失败')
    expect(wrapper.get('[data-testid="server-startup-error-banner"]').text()).toContain('容器')
  })

  it('T4: SSE 未连接文案为未连接，aria 为启动状态 SSE 未连接', () => {
    const wrapper = mountPanel({
      isServerRunning: false,
      isServerStarting: false,
      serverStatus: 'stopped',
      sseLive: false,
    })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('已停止')
    expect(wrapper.get('[data-testid="sse-connection-status"]').text()).toContain('SSE 未连接')
    expect(
      wrapper.get('[data-testid="sse-connection-status"]').attributes('aria-label'),
    ).toBe('启动状态 SSE 未连接')
  })

  it('T5: 空 statusMessage 不渲染文案节点（握手 message 已由后端置空）', () => {
    const wrapper = mountPanel({
      serverStatus: 'processing',
      statusMessage: '',
      statusProgress: 0,
    })
    expect(wrapper.find('[data-testid="server-start-status-message"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sse-connection-status"]').text()).toBe('SSE 已连接')
  })

  it('T6: 真实启动文案仍展示', () => {
    const wrapper = mountPanel({ statusMessage: '正在创建实例' })
    expect(wrapper.get('[data-testid="server-start-status-message"]').text()).toBe('正在创建实例')
  })

  it('T7: 冷打开仅 runtimeStatus=Running 时显示已启动', () => {
    const wrapper = mountPanel({
      serverStatus: '',
      isServerRunning: false,
      isServerStarting: false,
      runtimeStatus: 'Running',
      statusMessage: '',
      statusProgress: 0,
    })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('已启动')
  })

  it('T8: Runtime 面板不再展示容器连接状态（已迁入评论执行细节）', () => {
    const wrapper = mountPanel()
    expect(wrapper.text()).not.toContain('容器连接状态')
    expect(wrapper.find('[data-testid="container-connection-status"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sse-connection-status"]').exists()).toBe(true)
  })

  it('T9: 已停止 + SSE 重连时平台 hint 文案为推送通道不可用，且不写「平台服务重启中」', () => {
    const wrapper = mountPanel({
      serverStatus: 'stopped',
      isServerRunning: false,
      isServerStarting: false,
      sseLive: false,
      sseReconnecting: true,
      sseReconnectAttempts: 5,
      ssePlatformRestartHint: true,
      statusMessage: '',
      statusProgress: 0,
    })
    expect(wrapper.get('[data-testid="server-lifecycle-status"]').text()).toBe('已停止')
    expect(wrapper.get('[data-testid="sse-connection-status"]').text()).toContain('正在重连')
    const hint = wrapper.get('[data-testid="sse-platform-restart-hint"]')
    expect(hint.text()).toBe('状态推送服务暂不可用')
    expect(wrapper.text()).not.toContain('平台服务重启中')
  })

  it('T10: sseLive 时不展示平台推送 hint', () => {
    const wrapper = mountPanel({
      sseLive: true,
      ssePlatformRestartHint: true,
    })
    expect(wrapper.find('[data-testid="sse-platform-restart-hint"]').exists()).toBe(false)
  })

  it('T11: startTraceId 挂载到面板 data-traceId', () => {
    const wrapper = mountPanel({
      serverStatus: 'error',
      isServerRunning: false,
      statusMessage: '未找到匹配地域的运行环境',
      startTraceId: 'task_15652393783064603866',
    })
    const panel = wrapper.get('[data-testid="server-start-status-panel"]')
    expect(panel.attributes('data-traceid') || panel.attributes('data-traceId')).toBe(
      'task_15652393783064603866',
    )
  })

  it('T12: 空 startTraceId 时不挂 data-traceId', () => {
    const wrapper = mountPanel({ startTraceId: '' })
    const panel = wrapper.get('[data-testid="server-start-status-panel"]')
    expect(panel.attributes('data-traceid') || panel.attributes('data-traceId')).toBeUndefined()
  })

  it('T13: 释放后仅有 statusLogs 时仍展示启动日志', () => {
    const wrapper = mountPanel({
      serverStatus: '',
      isServerRunning: false,
      isServerStarting: false,
      runtimeStatus: '',
      statusMessage: '',
      statusLogs: [
        '[06:00:14] 容器调度排队中',
        '[06:05:02] 正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）',
      ],
    })
    expect(wrapper.get('[data-testid="server-start-status-panel"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('容器调度排队中')
    expect(wrapper.text()).toContain('触发：容器指令空闲超时回收')
  })

  it('T14: COS hydrate 形态的后端 logs 仍渲染「启动日志」标题与正文', () => {
    const wrapper = mountPanel({
      serverStatus: '',
      isServerRunning: false,
      isServerStarting: false,
      runtimeStatus: '',
      statusMessage: '',
      statusLogs: ['[02:00:00] 正在启动容器实例'],
    })
    const heading = wrapper.findAll('h4').find((n) => n.text() === '启动日志')
    expect(heading).toBeTruthy()
    expect(heading.classes()).toContain('text-xs')
    expect(wrapper.text()).toContain('正在启动容器实例')
  })
})
}
