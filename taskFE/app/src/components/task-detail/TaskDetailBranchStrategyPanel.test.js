// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] TaskDetailBranchStrategyPanel.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mount } = await import('@vue/test-utils')
  const { default: TaskDetailBranchStrategyPanel } = await import('./TaskDetailBranchStrategyPanel.vue')

describe('TaskDetailBranchStrategyPanel merged inputs', () => {
  const editingTask = {
    id: 'task_1',
    title: 'hello',
    workBranchPreset: 'feature',
    workBranchName: 'feature/2026-07-12_tester_daydaymoneytask_1_hello',
    mergeTargetPreset: 'custom',
    mergeTargetName: 'develop',
  }

  const mountEditing = (overrides = {}) =>
    mount(TaskDetailBranchStrategyPanel, {
      props: {
        task: {
          title: 'hello',
          created_at: '2026-07-12T00:00:00Z',
          branch_strategy: {
            work_branch_name: editingTask.workBranchName,
            merge_target_branch_name: 'develop',
          },
        },
        taskId: 'task_1',
        isEditing: true,
        editingTask: { ...editingTask },
        companyUserName: 'tester',
        commonMergeTargetBranches: ['develop', 'main'],
        commonMergeTargetBranchesLoading: false,
        commonMergeTargetBranchesError: '',
        ...overrides,
      },
    })

  it('T1: 编辑态只有工作分支与目标分支两个输入，无模板 select', () => {
    const wrapper = mountEditing()
    expect(wrapper.find('#edit-work-branch').exists()).toBe(true)
    expect(wrapper.find('#edit-merge-target-branch').exists()).toBe(true)
    expect(wrapper.find('#edit-work-branch-preset').exists()).toBe(false)
    expect(wrapper.find('#edit-merge-target-preset').exists()).toBe(false)
    expect(wrapper.findAll('select').length).toBe(0)
    expect(wrapper.findAll('input[type="text"]').length).toBe(2)
  })

  it('T2: 工作分支 datalist 含模版候选', () => {
    const wrapper = mountEditing()
    const values = wrapper.findAll('#edit-work-branch-preset-options option').map((n) => n.attributes('value'))
    expect(values.some((v) => String(v).startsWith('feature/'))).toBe(true)
    expect(values.some((v) => String(v).startsWith('bugfix/'))).toBe(true)
  })

  it('T3: 目标分支 datalist 为共有分支', () => {
    const wrapper = mountEditing()
    const values = wrapper.findAll('#edit-merge-target-common-options option').map((n) => n.attributes('value'))
    expect(values).toEqual(['develop', 'main'])
  })

  it('T4: 手改工作分支为非模版值时 preset 变为 custom', async () => {
    const wrapper = mountEditing()
    const input = wrapper.find('#edit-work-branch')
    await input.setValue('my/custom-branch')
    await input.trigger('input')
    expect(wrapper.props('editingTask').workBranchPreset).toBe('custom')
  })
})
}
