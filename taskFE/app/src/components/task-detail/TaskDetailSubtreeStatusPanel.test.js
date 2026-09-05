// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailSubtreeStatusPanel.test.js requires vitest runtime')
} else {
const { describe, it, expect } = await import('vitest')
const { mount } = await import('@vue/test-utils')
const { createRouter, createMemoryHistory } = await import('vue-router')
const { default: TaskDetailSubtreeStatusPanel } = await import('./TaskDetailSubtreeStatusPanel.vue')

async function mountPanel(props) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/t/:tenant/w/:workspaceId/task/:taskId', name: 'task_detail', component: { template: '<div/>' } }],
  })
  await router.push('/')
  return mount(TaskDetailSubtreeStatusPanel, {
    props: {
      tenantId: 't1',
      workspaceId: 'w1',
      ...props,
    },
    global: { plugins: [router] },
  })
}

describe('TaskDetailSubtreeStatusPanel', () => {
  it('无子树时不渲染', async () => {
    const wrapper = await mountPanel({ summary: { total: 0, settled: 0 }, nodes: [] })
    expect(wrapper.find('[data-testid="task-subtree-status"]').exists()).toBe(false)
  })

  it('展示摘要与节点状态', async () => {
    const wrapper = await mountPanel({
      summary: { total: 2, settled: 1 },
      nodes: [
        { id: 'c1', title: '子任务', depth: 1, progress_column_name: '进行中', settled: false },
        { id: 'g1', title: '孙任务', depth: 2, progress_column_name: '已完成', settled: true },
      ],
    })
    expect(wrapper.find('[data-testid="task-subtree-summary"]').text()).toContain('已关闭 1/2')
    expect(wrapper.find('[data-testid="task-subtree-node-c1"]').text()).toContain('进行中')
    expect(wrapper.find('[data-testid="task-subtree-node-g1"]').classes().join(' ')).toContain('pl-4')
  })
})

}
