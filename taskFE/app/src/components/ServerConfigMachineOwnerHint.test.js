// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] ServerConfigMachineOwnerHint.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { default: ServerConfigMachineOwnerHint } = await import('./ServerConfigMachineOwnerHint.vue')

describe('ServerConfigMachineOwnerHint', () => {
  it('容器未运行时不渲染', () => {
    const wrapper = mount(ServerConfigMachineOwnerHint, {
      props: {
        runtimeStatus: {
          container_running: false,
          instance_id: 'i-1',
          machine_owner_task_ids: ['task-a'],
        },
        viewerTaskId: 'task-a',
        tenantId: 't1',
        workspaceId: 'w1',
      },
    })
    expect(wrapper.find('[data-testid="machine-owner-hint"]').exists()).toBe(false)
  })

  it('本任务独占时展示本任务标注', () => {
    const wrapper = mount(ServerConfigMachineOwnerHint, {
      props: {
        runtimeStatus: {
          container_running: true,
          instance_id: 'i-1',
          machine_owner_task_ids: ['task-a'],
        },
        viewerTaskId: 'task-a',
        tenantId: 't1',
        workspaceId: 'w1',
      },
    })
    const hint = wrapper.get('[data-testid="machine-owner-hint"]')
    expect(hint.text()).toContain('机器节点所属任务')
    expect(hint.text()).toContain('task-a')
    expect(hint.text()).toContain('本任务')
    expect(wrapper.find('[data-testid="machine-owner-task-link-task-a"]').exists()).toBe(false)
  })

  it('共享实例时列出他任务链接', () => {
    const wrapper = mount(ServerConfigMachineOwnerHint, {
      props: {
        runtimeStatus: {
          container_running: true,
          instance_id: 'i-shared',
          machine_owner_task_ids: ['task-a', 'task-b'],
        },
        viewerTaskId: 'task-a',
        tenantId: 't1',
        workspaceId: 'w1',
      },
    })
    expect(wrapper.get('[data-testid="machine-owner-hint"]').text()).toContain('task-b')
    const link = wrapper.get('[data-testid="machine-owner-task-link-task-b"]')
    expect(link.attributes('href')).toBe('/tenant/t1/workspace/w1/task-detail/task-b/')
  })
})

}
