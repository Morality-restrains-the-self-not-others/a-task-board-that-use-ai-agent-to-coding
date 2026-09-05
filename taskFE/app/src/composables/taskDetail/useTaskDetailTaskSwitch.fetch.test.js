// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] useTaskDetailTaskSwitch.fetch.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')

  describe('useTaskDetailTaskSwitch fetchTaskDetail', () => {
    it('切换任务时始终拉取详情评论 feed，不因 props.task 存在而跳过', () => {
      const here = dirname(fileURLToPath(import.meta.url))
      const src = readFileSync(join(here, 'useTaskDetailTaskSwitch.js'), 'utf8')
      expect(src).toContain('$.fetchWorkspaceProjects(), $.fetchTaskDetail()')
      expect(src).not.toMatch(/if\s*\(\s*!props\.task\s*\)\s*jobs\.push\(\$\.fetchTaskDetail\(\)\)/)
    })
  })

  describe('TaskDetail.vue mount fetchTaskDetail', () => {
    it('工作面板传入 props.task 时仍拉取详情（含评论 feed）', () => {
      const here = dirname(fileURLToPath(import.meta.url))
      const src = readFileSync(join(here, '../../views/TaskDetail.vue'), 'utf8')
      expect(src).toContain('initTasks.push(fetchTaskDetail())')
      expect(src).not.toMatch(/if\s*\(\s*!props\.task\s*\)\s*initTasks\.push\(fetchTaskDetail\(\)\)/)
    })
  })
}
