import { describe, expect, it } from 'vitest'
import {
  countTodosByKanbanColumns,
  filterTodosByKanbanColumn,
  mapProgressColumnsToTaskStatuses,
  readTodoProgressColumnId,
  resolveTodoKanbanColumnId,
  sortProgressColumns,
  sortTodosForKanbanColumn,
} from './workPanelKanbanUtils.js'

describe('workPanelKanbanUtils', () => {
  const statuses = [
    { id: '1000000000000000101', name: '待处理' },
    { id: '1000000000000000104', name: '已取消' },
  ]

  it('reads progress_column_id from API shape', () => {
    expect(readTodoProgressColumnId({ progress_column_id: '1000000000000000104' })).toBe(
      '1000000000000000104',
    )
  })

  it('maps null progress_column_id to first column for kanban display', () => {
    expect(resolveTodoKanbanColumnId({ progress_column_id: null }, statuses)).toBe(
      '1000000000000000101',
    )
  })

  it('places tasks in matching column when progress_column_id is set', () => {
    const todos = [
      { id: '1', progress_column_id: '1000000000000000104', title: 'cancelled' },
      { id: '2', progress_column_id: null, title: 'unassigned' },
    ]
    expect(filterTodosByKanbanColumn(todos, '1000000000000000104', statuses).map((t) => t.id)).toEqual([
      '1',
    ])
    expect(filterTodosByKanbanColumn(todos, '1000000000000000101', statuses).map((t) => t.id)).toEqual([
      '2',
    ])
  })

  it('returns null when progress-system columns are empty', () => {
    expect(mapProgressColumnsToTaskStatuses([])).toBeNull()
    expect(mapProgressColumnsToTaskStatuses(null)).toBeNull()
  })

  it('falls back unknown progress_column_id to first column when statuses use snowflake ids', () => {
    expect(
      resolveTodoKanbanColumnId({ progress_column_id: '999' }, statuses),
    ).toBe('1000000000000000101')
  })

  it('sorts progress columns by order', () => {
    const mapped = mapProgressColumnsToTaskStatuses([
      { id: '2', name: 'B', order: '2' },
      { id: '1', name: 'A', order: '1' },
    ])
    expect(mapped?.map((s) => s.name)).toEqual(['A', 'B'])
  })

  it('sortProgressColumns sorts by order_num with id fallback (shared helper)', () => {
    expect(
      sortProgressColumns([
        { id: 'col-wip', name: '进行中', order_num: 1 },
        { id: 'col-todo', name: '待处理', order_num: 0 },
      ]).map((c) => c.id),
    ).toEqual(['col-todo', 'col-wip'])
    expect(
      sortProgressColumns([
        { id: '2', name: 'B', order: '2' },
        { id: '1', name: 'A', order: '1' },
      ]).map((c) => c.id),
    ).toEqual(['1', '2'])
    expect(sortProgressColumns(null)).toEqual([])
  })

  it('sorts column tasks by order ascending so smaller order appears first', () => {
    const sorted = sortTodosForKanbanColumn([
      { id: '1', order: '0', title: 'older-top-after-drag' },
      { id: '2', order: '-1', title: 'newly-created' },
      { id: '3', order: '1', title: 'bottom' },
    ])
    expect(sorted.map((t) => t.id)).toEqual(['2', '1', '3'])
  })

  it('counts todos per progress column in status order', () => {
    const statusesWithColor = [
      { id: '1000000000000000101', name: '待处理', color: '#ff6b6b' },
      { id: '1000000000000000104', name: '已取消', color: '#6c757d' },
    ]
    const todos = [
      { id: '1', progress_column_id: '1000000000000000104' },
      { id: '2', progress_column_id: null },
      { id: '3', progress_column_id: '1000000000000000101' },
      { id: '4', progress_column_id: '1000000000000000104' },
    ]
    expect(countTodosByKanbanColumns(todos, statusesWithColor)).toEqual([
      { id: '1000000000000000101', name: '待处理', color: '#ff6b6b', count: 2 },
      { id: '1000000000000000104', name: '已取消', color: '#6c757d', count: 2 },
    ])
  })

  it('returns empty column counts when statuses missing', () => {
    expect(countTodosByKanbanColumns([{ id: '1' }], null)).toEqual([])
    expect(countTodosByKanbanColumns([], [])).toEqual([])
  })
})
