import { describe, expect, it } from 'vitest'
import {
  linkedProjectRowsFromTask,
  unwrapTaskDetailPayload,
} from './unwrapTaskDetailPayload.js'

const TASK_ID = 'task_881388002226499584'
const PROJECT_ROW = {
  project_id: 'proj_881195029417193472',
  project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
  stored_repo_address: 'https://github.com/ruandao/helloworld',
  base_branch: 'master',
}

describe('unwrapTaskDetailPayload', () => {
  it('returns a matching task object unchanged', () => {
    const raw = { id: TASK_ID, title: '用 js 写一个 hello world', projects: [PROJECT_ROW] }
    expect(unwrapTaskDetailPayload(raw, TASK_ID)).toBe(raw)
  })

  it('picks the matching task when GET was misrouted to a workspace list', () => {
    const wanted = { id: TASK_ID, projects: [PROJECT_ROW] }
    const raw = [
      { id: 'task_other', projects: [] },
      wanted,
    ]
    expect(unwrapTaskDetailPayload(raw, TASK_ID)).toBe(wanted)
  })

  it('unwraps { data: task } envelopes', () => {
    const task = { id: TASK_ID, projects: [PROJECT_ROW] }
    expect(unwrapTaskDetailPayload({ data: task }, TASK_ID)).toBe(task)
  })

  it('returns null when a list does not contain the requested task', () => {
    expect(unwrapTaskDetailPayload([{ id: 'task_other', projects: [] }], TASK_ID)).toBeNull()
  })

  it('returns null for a non-object payload', () => {
    expect(unwrapTaskDetailPayload('not-json', TASK_ID)).toBeNull()
  })
})

describe('linkedProjectRowsFromTask', () => {
  it('reads projects from a task record', () => {
    expect(linkedProjectRowsFromTask({ id: TASK_ID, projects: [PROJECT_ROW] }, TASK_ID)).toEqual([PROJECT_ROW])
  })

  it('reads projects when localTask was wrongly assigned a workspace list', () => {
    const rows = linkedProjectRowsFromTask(
      [
        { id: 'task_other', projects: [] },
        { id: TASK_ID, projects: [PROJECT_ROW] },
      ],
      TASK_ID,
    )
    expect(rows).toEqual([PROJECT_ROW])
  })
})
