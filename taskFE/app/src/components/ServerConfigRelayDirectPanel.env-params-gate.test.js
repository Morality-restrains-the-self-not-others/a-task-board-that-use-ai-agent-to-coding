// @vitest-environment jsdom
/**
 * ServerConfigRelayDirectPanel：智能体资源配置未选时禁用启动
 */
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] ServerConfigRelayDirectPanel.env-params-gate.test.js requires vitest runtime')
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
      ...props,
    },
  })
}

describe('ServerConfigRelayDirectPanel env params gate', () => {
  it('智能体资源配置未选时禁用启动并展示提示', () => {
    const wrapper = mountPanel({
      envParamsSourceRequiredHint: '请先选择智能体资源配置',
    })
    const startBtn = wrapper.find('[data-testid="relay-to-trae-start-btn"]')
    expect(startBtn.attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="relay-to-trae-start-disabled-hint"]').text())
      .toBe('请先选择智能体资源配置')
  })

  it('已选智能体资源配置且其余条件满足时可启动', () => {
    const wrapper = mountPanel({
      envParamsSourceRequiredHint: '',
    })
    const startBtn = wrapper.find('[data-testid="relay-to-trae-start-btn"]')
    expect(startBtn.attributes('disabled')).toBeUndefined()
    expect(wrapper.find('[data-testid="relay-to-trae-start-disabled-hint"]').exists()).toBe(false)
  })
})
}
