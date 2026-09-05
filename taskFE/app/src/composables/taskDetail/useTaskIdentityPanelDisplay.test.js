// @vitest-environment jsdom
/**
 * useTaskIdentityPanelDisplay：当前任务人读编号 #N
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] useTaskIdentityPanelDisplay.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { computed } = await import('vue')
  const { useTaskIdentityPanelDisplay } = await import('./useTaskIdentityPanelDisplay.js')

  function setup(task, workspaceTodos = []) {
    return useTaskIdentityPanelDisplay(
      {
        task,
        collaboratorNameById: {},
        forkSourceTitle: '',
        forkSourceSeq: 0,
        isForkSourceTitleLoading: false,
        workspaceTodos,
      },
      {
        resolvedParentTaskTitle: computed(() => ''),
        resolvedParentTaskSeq: computed(() => 0),
        parentDeliverableId: computed(() => null),
        workspaceTodos: computed(() => workspaceTodos),
      },
    )
  }

  describe('useTaskIdentityPanelDisplay taskDisplayNo', () => {
    it('优先用 task.workspace_seq 格式化为 #N', () => {
      const { taskDisplayNo } = setup({
        id: 'task_877131046670331904',
        workspace_seq: 3,
      })
      expect(taskDisplayNo.value).toBe('#3')
    })

    it('详情缺序号时回退 workspaceTodos 中同 id 的 workspace_seq', () => {
      const { taskDisplayNo } = setup(
        { id: 'task_a', workspace_seq: 0 },
        [{ id: 'task_a', workspace_seq: 12 }],
      )
      expect(taskDisplayNo.value).toBe('#12')
    })

    it('无有效序号时为空串（不回退技术 ID）', () => {
      const { taskDisplayNo } = setup({ id: 'task_a' })
      expect(taskDisplayNo.value).toBe('')
    })
  })
}
