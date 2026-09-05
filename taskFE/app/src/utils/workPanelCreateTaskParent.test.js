// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  appendParentTaskToCreatePayload,
  buildCreateTaskDraft,
  resolveParentDeliverableBlockedReason,
  buildInitialParentTaskId,
  resolvePreferredParentTaskIdFromFilterBars,
} from './workPanelCreateTaskParent.js'

const cats = [
  { id: 'c1', name: '价值流', order: 1 },
  { id: 'c2', name: '业务流程', order: 2 },
]

describe('workPanelCreateTaskParent', () => {
  it('resolves preferred parent from filter bars', () => {
    const path = [
      { type: 'root', id: 'root', label: '全部' },
      { type: 'task', id: 't-root', label: '根' },
    ]
    expect(resolvePreferredParentTaskIdFromFilterBars([{ path }])).toBe('t-root')
    expect(resolvePreferredParentTaskIdFromFilterBars([])).toBe('')
  })

  it('buildInitialParentTaskId only for non-top category', () => {
    const todos = [{ id: 't-root', title: '根', deliverable_obj_id: 'c1' }]
    expect(buildInitialParentTaskId({
      taskTypes: cats, todos, defaultTypeId: 'c1', preferredParentTaskId: 't-root',
    })).toBe('')
    expect(buildInitialParentTaskId({
      taskTypes: cats, todos, defaultTypeId: 'c2', preferredParentTaskId: 't-root',
    })).toBe('t-root')
  })

  it('appendParentTaskToCreatePayload clears top-level and sets non-top', () => {
    const payload = {}
    appendParentTaskToCreatePayload(payload, { task_type: { id: 'c1' }, parent_task: 'x' }, cats)
    expect(payload.parent_task).toBe('')
    appendParentTaskToCreatePayload(payload, { task_type: { id: 'c2' }, parent_task: 't-root' }, cats)
    expect(payload.parent_task).toBe('t-root')
    appendParentTaskToCreatePayload(payload, { deliverable_obj_id: 'c2', parent_task: 't-root' }, cats)
    expect(payload.parent_task).toBe('t-root')
  })

  it('resolveParentDeliverableBlockedReason requires parent for non-top', () => {
    const todos = [{ id: 'p1', title: 'root', deliverable_obj_id: 'c1' }]
    expect(resolveParentDeliverableBlockedReason({
      taskTypes: cats, todos, categoryId: 'c1', parentTaskId: '',
    })).toBe('')
    expect(resolveParentDeliverableBlockedReason({
      taskTypes: cats, todos, categoryId: 'c2', parentTaskId: '',
    })).toContain('上层交付物')
    expect(resolveParentDeliverableBlockedReason({
      taskTypes: cats, todos, categoryId: 'c2', parentTaskId: 'p1',
    })).toBe('')
  })

  it('buildCreateTaskDraft prefers last-viewed project over list order', () => {
    const draft = buildCreateTaskDraft({
      projects: [{ id: 'proj_aaa_first' }, { id: 'proj_882824768007467008' }],
      preferredProjectId: 'proj_882824768007467008',
    })
    expect(draft.projectSelections[0].projectId).toBe('proj_882824768007467008')
  })

  it('buildCreateTaskDraft defaults owner and operator from opts', () => {
    const draft = buildCreateTaskDraft({
      ownerId: 'uid-owner',
      operatorId: 'uid-operator',
    })
    expect(draft.owner).toBe('uid-owner')
    expect(draft.operator).toBe('uid-operator')
    expect(draft.queued_auto_run).toBe(false)
  })
})
