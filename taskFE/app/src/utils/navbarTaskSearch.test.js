// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] navbarTaskSearch.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    buildTaskDetailHref,
    buildTaskSearchQuery,
    buildWorkPanelHref,
    findTodoByTaskIdQuery,
    formatSearchHitLabel,
    matchMemberIdsByQuery,
    normalizeSearchText,
  } = await import('./navbarTaskSearch.js')

  describe('navbarTaskSearch', () => {
    it('normalizeSearchText 还原 task-task_ 前缀与 #', () => {
      expect(normalizeSearchText('  task-task_13838391081043321882  ')).toBe(
        'task_13838391081043321882',
      )
      expect(normalizeSearchText('#task_13838391081043321882')).toBe(
        'task_13838391081043321882',
      )
      expect(normalizeSearchText('321882')).toBe('321882')
      expect(normalizeSearchText('Alpha')).toBe('Alpha')
    })

    it('normalizeSearchText 从评论容器名还原任务 id', () => {
      expect(normalizeSearchText('task_877071722828820480_cmt_877071748669927424')).toBe(
        'task_877071722828820480',
      )
      expect(normalizeSearchText('#task_877071722828820480_cmt_877071748669927424')).toBe(
        'task_877071722828820480',
      )
      expect(normalizeSearchText('task-task_877071722828820480_cmt_877071748669927424')).toBe(
        'task_877071722828820480',
      )
      expect(normalizeSearchText('task_task_877071722828820480_cmt_877071748669927424')).toBe(
        'task_877071722828820480',
      )
      expect(normalizeSearchText('cmt_877071748669927424')).toBe('cmt_877071748669927424')
      expect(normalizeSearchText('Alpha_cmt_not_an_id')).toBe('Alpha_cmt_not_an_id')
    })

    it('matchMemberIdsByQuery 按名称模糊匹配', () => {
      const members = [
        { id: 'm1', name: '张三' },
        { id: 'm2', name: '李四' },
        { id: 'm3', name: 'Alice' },
      ]
      expect(matchMemberIdsByQuery(members, '张')).toEqual(['m1'])
      expect(matchMemberIdsByQuery(members, 'ali')).toEqual(['m3'])
      expect(matchMemberIdsByQuery(members, '')).toEqual([])
    })

    it('buildTaskSearchQuery 含 q 与 assignee_ids', () => {
      const qs = buildTaskSearchQuery({ q: 'foo', assigneeIds: ['a', 'b'], limit: 10 })
      const p = new URLSearchParams(qs)
      expect(p.get('q')).toBe('foo')
      expect(p.get('assignee_ids')).toBe('a,b')
      expect(p.get('limit')).toBe('10')
    })

    it('buildWorkPanelHref 带 workspace 与 task_id', () => {
      expect(buildWorkPanelHref('850', { workspaceId: 'ws1', taskId: 't9' })).toBe(
        '/tenant/850/work-panel/?workspace_id=ws1&task_id=t9',
      )
    })

    it('buildTaskDetailHref 指向独立任务详情页且不同于当前工作面板 URL', () => {
      const detail = buildTaskDetailHref('850', { workspaceId: 'ws1', taskId: 'task-1' })
      expect(detail).toBe('/tenant/850/workspace/ws1/task-detail/task-1/')
      expect(detail).not.toBe(buildWorkPanelHref('850', { workspaceId: 'ws1', taskId: 'task-1' }))
    })

    it('buildTaskDetailHref 携带 comment_id 时附加 comment 查询参数（OPT-20260817-013）', () => {
      const detail = buildTaskDetailHref('850', {
        workspaceId: 'ws1',
        taskId: 'task-1',
        commentId: 'cmt_123',
      })
      expect(detail).toBe('/tenant/850/workspace/ws1/task-detail/task-1/?comment=cmt_123')
    })

    it('buildTaskDetailHref comment 与 accessCode 共存时保留两个参数', () => {
      const detail = buildTaskDetailHref('850', {
        workspaceId: 'ws1',
        taskId: 'task-1',
        accessCode: 'abc',
        commentId: 'cmt_123',
      })
      expect(detail).toBe('/tenant/850/workspace/ws1/task-detail/task-1/?accessCode=abc&comment=cmt_123')
    })

    it('findTodoByTaskIdQuery 支持完整 id 与 workspace_seq', () => {
      const todos = [{ id: 'abc958867', workspace_seq: 12 }, { id: 'other', workspace_seq: 3 }]
      expect(findTodoByTaskIdQuery(todos, 'abc958867')?.id).toBe('abc958867')
      expect(findTodoByTaskIdQuery(todos, '12')?.id).toBe('abc958867')
      expect(findTodoByTaskIdQuery(todos, '#3')?.id).toBe('other')
    })

    it('findTodoByTaskIdQuery 支持 task-task_ 误粘贴前缀', () => {
      const todos = [{ id: 'task_13838391081043321882' }, { id: 'other' }]
      expect(findTodoByTaskIdQuery(todos, 'task-task_13838391081043321882')?.id).toBe(
        'task_13838391081043321882',
      )
    })

    it('buildTaskSearchQuery 对 task-task_ 前缀做规范化', () => {
      const qs = buildTaskSearchQuery({ q: 'task-task_13838391081043321882', limit: 20 })
      const p = new URLSearchParams(qs)
      expect(p.get('q')).toBe('task_13838391081043321882')
    })

    it('buildTaskSearchQuery 对评论容器名做规范化', () => {
      const qs = buildTaskSearchQuery({
        q: 'task_877071722828820480_cmt_877071748669927424',
        limit: 20,
      })
      const p = new URLSearchParams(qs)
      expect(p.get('q')).toBe('task_877071722828820480')
    })

    it('findTodoByTaskIdQuery 支持评论容器名', () => {
      const todos = [{ id: 'task_877071722828820480' }, { id: 'other' }]
      expect(findTodoByTaskIdQuery(todos, 'task_877071722828820480_cmt_877071748669927424')?.id).toBe(
        'task_877071722828820480',
      )
    })

    it('formatSearchHitLabel 展示编号、标题、负责人与操作员', () => {
      const label = formatSearchHitLabel(
        { id: 'xx958867', workspace_seq: 12, title: 'Demo', owner: 'm1', operator: 'm3', assignees: ['m2'] },
        new Map([['m1', 'Alice'], ['m2', 'Bob'], ['m3', 'Dave']]),
      )
      expect(label).toContain('#12')
      expect(label).toContain('Demo')
      expect(label).toContain('Alice')
      expect(label).toContain('Bob')
      expect(label).toContain('Dave')
    })
  })
}
