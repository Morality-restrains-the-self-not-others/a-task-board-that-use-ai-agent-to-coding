// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] WorkspaceSettingsTaskPanelActions.unit.test.js requires vitest runtime')
} else {
  const { mount } = await import('@vue/test-utils')
  const { describe, expect, it } = await import('vitest')
  const { default: WorkspaceSettingsTaskPanelActions } = await import('./WorkspaceSettingsTaskPanelActions.vue')

describe('WorkspaceSettingsTaskPanelActions', () => {
  it('exposes create-fields entry and does not expose standalone task-kind/code-lang buttons', () => {
    const wrapper = mount(WorkspaceSettingsTaskPanelActions, {
      props: {
        workspace: { value: 'ws-1', text: 'Demo' },
      },
    })
    expect(wrapper.find('[data-testid="open-create-task-field-settings"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="open-task-kind-options"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="open-code-lang-options"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('创建字段')
    expect(wrapper.text()).not.toContain('任务类型')
    expect(wrapper.text()).not.toContain('编程语言')
    wrapper.unmount()
  })

  it('labels the feature-params entry as 智能体资源配置 (renamed from 功能参数)', () => {
    const wrapper = mount(WorkspaceSettingsTaskPanelActions, {
      props: {
        workspace: { value: 'ws-1', text: 'Demo' },
      },
    })
    expect(wrapper.text()).toContain('智能体资源配置')
    expect(wrapper.text()).not.toContain('功能参数')
    wrapper.unmount()
  })

  it('套餐设置 button emits archive so the parent can open task-archive settings', async () => {
    const wrapper = mount(WorkspaceSettingsTaskPanelActions, {
      props: {
        workspace: { value: 'ws-1', text: 'Demo' },
      },
    })
    const btn = wrapper.find('[data-testid="open-task-archive-settings"]')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('套餐设置')
    await btn.trigger('click')
    expect(wrapper.emitted('archive')?.[0]?.[0]).toEqual({ value: 'ws-1', text: 'Demo' })
    wrapper.unmount()
  })
})
}
