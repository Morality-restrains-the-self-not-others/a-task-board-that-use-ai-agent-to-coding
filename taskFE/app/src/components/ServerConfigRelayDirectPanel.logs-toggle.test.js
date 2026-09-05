// @vitest-environment jsdom
/**
 * ServerConfigRelayDirectPanel：启动日志折叠/展开
 */
if (!process.env.VITEST) {
  console.log('[skip] ServerConfigRelayDirectPanel.logs-toggle.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: ServerConfigRelayDirectPanel } = await import('./ServerConfigRelayDirectPanel.vue')

function mountPanel(props = {}) {
  return mount(ServerConfigRelayDirectPanel, {
    props: {
      visible: true,
      hasTaskId: true,
      hasImage: true,
      envItems: [],
      logs: ['line-a', 'line-b'],
      ...props,
    },
  })
}

describe('ServerConfigRelayDirectPanel logs toggle', () => {
  it('默认折叠时隐藏日志正文，按钮文案为展开日志', () => {
    const wrapper = mountPanel()
    const toggle = wrapper.find('[data-testid="relay-to-trae-logs-toggle"]')
    expect(toggle.exists()).toBe(true)
    expect(toggle.text()).toBe('展开日志')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-testid="relay-to-trae-logs"]').isVisible()).toBe(false)
  })

  it('logsExpanded=true 时展示日志正文，按钮文案为折叠日志', () => {
    const wrapper = mountPanel({ logsExpanded: true })
    const toggle = wrapper.find('[data-testid="relay-to-trae-logs-toggle"]')
    expect(toggle.text()).toBe('折叠日志')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    const logs = wrapper.find('[data-testid="relay-to-trae-logs"]')
    expect(logs.isVisible()).toBe(true)
    expect(logs.text()).toContain('line-a')
  })

  it('点击切换按钮发出 toggleLogs', async () => {
    const wrapper = mountPanel({ logsExpanded: true })
    await wrapper.find('[data-testid="relay-to-trae-logs-toggle"]').trigger('click')
    expect(wrapper.emitted('toggleLogs')).toHaveLength(1)
  })

  it('折叠时引导状态条仍可见', () => {
    const wrapper = mountPanel({
      logsExpanded: false,
      logs: ['[onlineServiceJS] BOOTSTRAP_PHASE=task_detail_begin 容器已启动，开始拉取任务详情…'],
    })
    expect(wrapper.find('[data-testid="relay-to-trae-bootstrap-status"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="relay-to-trae-logs"]').isVisible()).toBe(false)
  })
})

}
