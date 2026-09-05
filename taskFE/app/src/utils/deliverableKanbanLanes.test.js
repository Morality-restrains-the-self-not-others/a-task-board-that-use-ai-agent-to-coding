// @vitest-environment node
import { describe, it, expect } from 'vitest'
import { filterTodosByKanbanColumn } from './workPanelKanbanUtils.js'

/**
 * 纵轴=进度列：每个进度纵轴只含该进度的任务。
 */
function tasksInProgressColumn(todos, statusId, taskStatuses) {
  return filterTodosByKanbanColumn(todos, statusId, taskStatuses)
}

describe('progress columns (one progress per vertical axis)', () => {
  const statuses = [
    { id: 's1', name: '待办' },
    { id: 's2', name: '进行中' },
  ]
  const todos = [
    { id: 't1', deliverable_obj_id: 'd1', progress_column_id: 's1' },
    { id: 't2', deliverable_obj_id: 'd1', progress_column_id: 's2' },
    { id: 't3', deliverable_obj_id: 'd2', progress_column_id: 's1' },
  ]

  it('each progress column only contains that progress', () => {
    expect(tasksInProgressColumn(todos, 's1', statuses).map((t) => t.id).sort()).toEqual([
      't1',
      't3',
    ])
    expect(tasksInProgressColumn(todos, 's2', statuses).map((t) => t.id)).toEqual(['t2'])
  })
})
