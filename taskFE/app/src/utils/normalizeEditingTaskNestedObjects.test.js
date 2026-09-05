// @vitest-environment node
/**
 * pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
 */
if (!process.env.VITEST) {
  console.log('[skip] normalizeEditingTaskNestedObjects.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { normalizeEditingTaskNestedObjects } = await import('./normalizeEditingTaskNestedObjects.js')

  describe('normalizeEditingTaskNestedObjects deliverable category', () => {
    it('从 deliverable_obj_id 回填 task_type.id', () => {
      const task = { deliverable_obj_id: 'd9' }
      normalizeEditingTaskNestedObjects(task, {
        taskTypes: [{ id: 'd1', name: '价值流' }],
      })
      expect(task.task_type.id).toBe('d9')
    })

    it('task_type 缺失 id 时用 deliverable_obj_id 补齐', () => {
      const task = { deliverable_obj_id: 'd8', task_type: { id: null } }
      normalizeEditingTaskNestedObjects(task, {
        taskTypes: [{ id: 'd1', name: '价值流' }],
      })
      expect(task.task_type.id).toBe('d8')
    })

    it('无 deliverable 时回退第一个 taskTypes', () => {
      const task = {}
      normalizeEditingTaskNestedObjects(task, {
        taskTypes: [{ id: 'd1', name: '价值流' }],
      })
      expect(task.task_type.id).toBe('d1')
    })
  })
}
