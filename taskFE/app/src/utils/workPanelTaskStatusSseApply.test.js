import { describe, it, expect } from 'vitest'
import { applyWorkPanelTaskStatusChanged } from './workPanelTaskStatusSseApply.js'

describe('applyWorkPanelTaskStatusChanged', () => {
  it('updates progress_column_id when task matches', () => {
    const todos = [
      { id: 't1', progress_column_id: 'col-a', completed: false },
      { id: 't2', progress_column_id: 'col-b', completed: false },
    ]
    const { todos: next, matched, changed } = applyWorkPanelTaskStatusChanged(todos, {
      event_name: 'task_status_changed',
      task_id: 't1',
      progress_column_id: 'col-done',
      completed: false,
    })
    expect(matched).toBe(true)
    expect(changed).toBe(true)
    expect(next[0].progress_column_id).toBe('col-done')
    expect(next[1].progress_column_id).toBe('col-b')
  })

  it('ignores heartbeat-like payloads without task_id', () => {
    const todos = [{ id: 't1', progress_column_id: 'col-a' }]
    const { matched, changed, todos: next } = applyWorkPanelTaskStatusChanged(todos, {
      type: 'heartbeat',
      event_name: 'work_panel_sse_heartbeat',
    })
    expect(matched).toBe(false)
    expect(changed).toBe(false)
    expect(next).toBe(todos)
  })

  it('returns matched false when task not in list', () => {
    const todos = [{ id: 't1', progress_column_id: 'col-a' }]
    const { matched, changed } = applyWorkPanelTaskStatusChanged(todos, {
      task_id: 'missing',
      progress_column_id: 'col-x',
    })
    expect(matched).toBe(false)
    expect(changed).toBe(false)
  })

  it('skips no-op when column unchanged', () => {
    const todos = [{ id: 't1', progress_column_id: 'col-a', completed: false }]
    const { changed, todos: next } = applyWorkPanelTaskStatusChanged(todos, {
      task_id: 't1',
      progress_column_id: 'col-a',
      completed: false,
    })
    expect(changed).toBe(false)
    expect(next).toBe(todos)
  })
})
